

package main

import (
	"github.com/robertkrimen/otto"
	_ "github.com/robertkrimen/otto/underscore"
)
import (
	"github.com/micro/go-config"
	"github.com/micro/go-config/source/file"
)
import "github.com/vjeantet/grok"
import "github.com/juliangruber/go-intersect"
import (
  "github.com/prometheus/client_golang/prometheus"
  "github.com/prometheus/client_golang/prometheus/promauto"
  "github.com/prometheus/client_golang/prometheus/promhttp"
)

import "encoding/json"
import "fmt"
import "time"
import "bufio"
import "os"
import "log"
import "context"
import "github.com/olivere/elastic"
import "io/ioutil"
import "net/http"


var steps = []string{}

type Message map[string]interface{}
type MessageW struct {
    msg *interface{}
}

type Output struct {
  Tags []string
  Msg  string
  Ts time.Time
}

type ESServer struct {
  Tags []string `yaml:"tags"`
  Name string   `yaml:"name"`
  Host string   `yaml:"host"`
}

type ESConfig struct {
  Servers []ESServer `yaml: "servers"`
}

var (
  logEventsTotalProcessed = promauto.NewCounter(prometheus.CounterOpts{
          Name: "logsmasher_events_count",
          Help: "The total number of processed events",
  })
  logEventsPassedProcessed = promauto.NewCounter(prometheus.CounterOpts{
          Name: "logsmasher_events_passed_count",
          Help: "The total number of processed events with passed pipeline",
  })
  ingressBatches = promauto.NewCounter(prometheus.CounterOpts{
          Name: "logsmasher_ingress_batches",
          Help: "The total number of batches ingested",
  })
)


func readSource(filename string) ([]byte, error) {
	if filename == "" || filename == "-" {
		return ioutil.ReadAll(os.Stdin)
	}
	return ioutil.ReadFile(filename)
}

func process_event(vm *otto.Otto, m Message) MessageW {
  var msg interface{}
  msg = m

  f, err := vm.Get("run_pipeline")
  if err != nil {
    log.Fatal(err)
  }
  // Convert message to JS-object for passing the steps:
  js_message, err := vm.ToValue(msg)
  if err != nil {
      log.Fatal(err)
  }
  // Call the pipeline
  result, err := f.Call(f, js_message);
  if err != nil {
    log.Fatal(err)
  }
  // Export the results back to Go
  rv, err := result.Export()
  if err != nil {
      log.Fatal(err)
  }

  return MessageW{msg: &rv}
}


func saveToQueue(ingress chan Output, conf ESConfig) {
  ctx := context.Background()
  esclients := map[string]*elastic.Client{}
  bps := map[string]*elastic.BulkProcessor{}

  log.Print("Elasticsearch fanout pool initializing")
  for _, server := range conf.Servers {
    log.Printf("Connecting to %s (%s) tags: %s\n", server.Host, server.Name, server.Tags)
    client, err := elastic.NewClient(elastic.SetURL(server.Host), elastic.SetSniff(false))
  	if err != nil {
  		// Handle error
  		log.Panic(err)
  	}

    log.Printf("Server %s connected!\n", server.Name)
    esclients[server.Name] = client
    p, err := client.BulkProcessor().
      Name(server.Name).
      Workers(2).
      BulkActions(1000).              // commit if # requests >= 1000
      BulkSize(2 << 20).              // commit if size of requests >= 2 MB
      FlushInterval(30*time.Second).  // commit every 30s
      Do(ctx)

    if err != nil {
      // Handle error
      log.Panic(err)
    }
    bps[server.Name] = p
  }

  log.Print("Elasticsearch fanout pool ready!")
  for message := range ingress {
    for _, server := range conf.Servers {
      common_tags := intersect.Hash(message.Tags, server.Tags).([]interface{})
      if len(common_tags) > 0 {
        indexName := fmt.Sprintf("logstash-%d.%02d.%02d",
          message.Ts.Year(), message.Ts.Month(), message.Ts.Day())
        r := elastic.NewBulkIndexRequest().
          Index(indexName).
          Type("fluentd").
          Doc(message.Msg)
        bps[server.Name].Add(r)
      }
    }
  }
}


func processEvents(ingress chan Message, egress chan Output) {
  vm := otto.New()

  log.Print("Event processor starting...")
  log.Print("Loading libraries.")
  libsrc, err := readSource("res/lib.js")
  if err != nil {
    log.Fatal("Library invalid!")
    log.Panic(err)
  }
  log.Print("Loading pipeline (pipeline.js).")
  src, err := readSource("config/pipeline.js")
  if err != nil {
    log.Fatal("Pipeline definiton invalid!")
    log.Panic(err)
  }

  log.Print("Registering helper functions.")
  vm.Set("print", func(call otto.FunctionCall) otto.Value {
    fmt.Printf("%s\n", call.Argument(0).String())
    return otto.Value{}
  })

  g, _ := grok.New()
  vm.Set("grok", func(call otto.FunctionCall) otto.Value {
    format := call.Argument(0).String()
    raw := call.Argument(1).String()

    values, err := g.Parse(format, raw)
    if err != nil {
      return otto.Value{}
    }
    res, _ := vm.ToValue(values)
    return res
  })

  log.Print("Evaluating libraries.")
  _, err = vm.Run(libsrc)
  if err != nil {
    log.Panic(err)
  }
  log.Print("Evaluating pipeline.")
  _, err = vm.Run(src)
  if err != nil {
    log.Panic(err)
  }

  log.Print("Event processor ready!")
  // Loop through messages
  for message := range ingress {
    // Metrics
    logEventsTotalProcessed.Inc()
    output := process_event(vm, message)
    if *output.msg != nil {
      logEventsPassedProcessed.Inc()
      msg := (*output.msg).(map[string]interface {})
      // Parse timestamp
      t, err := time.Parse(time.RFC3339, msg["ts"].(string))
      if err != nil {
          fmt.Println(err)
      }
      egress <- Output{
        Tags: msg["tags"].([]string),
        Ts: t,
        Msg: msg["msg"].(string),
      }
    }
  }
}

func takeEventBatch(egress chan Message) func(http.ResponseWriter, *http.Request) {
  return func(w http.ResponseWriter, r *http.Request) {
    ingressBatches.Inc()
    scanner := bufio.NewScanner(r.Body)
    for scanner.Scan() {
      var m Message
			jsonl := scanner.Text()
      json.Unmarshal([]byte(jsonl), &m)

      m["headers"] = map[string]string {
        "request_path": r.URL.Path,
      }
      egress<-m

    }

    if err := scanner.Err(); err != nil {
        log.Fatal(err)
    }
    w.Write([]byte("ack"))
  }
}



func main() {

  log.Print("LOGSMASHER STARTED")
  log.Print("Loading ES config (config.yml)")
  config.Load(file.NewSource(
  	file.WithPath("config/config.yml"),
  ))


  httpchan := make(chan Message)
  saverch := make(chan Output)
  var esconfig ESConfig
  config.Scan(&esconfig)

  go processEvents(httpchan, saverch)
  go saveToQueue(saverch, esconfig)

  http.Handle("/metrics", promhttp.Handler())
  http.HandleFunc("/", takeEventBatch(httpchan))
  log.Print("Listening at :8034")
  if err := http.ListenAndServe(":8034", nil); err != nil {
    panic(err)
  }


}
