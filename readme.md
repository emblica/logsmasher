# LOGSMASHER

Logstash like pipelines with Golang and Javascript

Supports Elasticsearch batch ingest protocol that most (Fluentd and logstash)
can use for ingesting events and acts as a middleware.


## Usage

2. Configure **pipeline.js**
    This file is used to configure the pipeline.
    It uses ES5 standard and supports some underscore.js commands.

2. Configure **config.yml**
    Add all Elasticsearch endpoints and tags for the events you want to ingest
    Note that single event can be ingested into multiple endpoints. (that might duplicate the data)
    Server is selected if **ANY** tag matches with any tag in event


```
docker run -it -v `pwd`/config:/config -p 8034:8034 docker.io/emblica/logsmasher
```


## Building

```
docker build -t logsmasher .
```

## Pipeline

Pipeline consists **steps**. Each step is executed after each other and passes the return value of previous steps.

All used steps must be declared in global variable `steps`

```js
steps = [
  step_a,
  step_b,
  step_c
]
```

### Step
All steps have signature: `f(message) -> message`
If step function returns `null` any following step won't be executed and the message will be dropped from the pipeline.

```js
// takes message as input
function calico (message) {
  // fetches container_name-field with helper function from the message and compares it
  if (container_name(message)=== "calico-node") {
    // If container_name was "calico-node" will add new tag "networking" to message.tags
    message.tags = add_tag(message, "networking")
  }
  // Return the message for next step
  return message
}
```

## Metrics

LOGSMASHER implements Prometheus-compatible metrics endpoint and provides some metrics along with Golang runtime specific metrics.

```
"logsmasher_events_count"
The total number of processed events

"logsmasher_events_passed_count"
The total number of processed events with passed pipeline

"logsmasher_ingress_batches"
The total number of batches ingested
```

## Kubernetes support

See manifests in `/k8s`

## License
See LICENSE
