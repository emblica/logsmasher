# LOGSMASHER

Logstash like pipelines with Golang and Javascript

Supports Elasticsearch batch ingest protocol that most (Fluentd and logstash)
can use for ingesting events and acts as middleware.


## Usage

2. Configure **pipeline.js**
    This file is used to configure the pipeline.
    It uses ES5 standard and supports some underscore.js commands.

2. Configure **config.yml**
    Add all Elasticsearch endpoints and tags for the events you want to ingest
    Note that single event can be ingested into multiple endpoints. (that might duplicate the data)
    Server is selected if **ANY** tag matches with any tag in event


```
docker run -it -v `pwd`/config:/config -p 8034:8034 logsmasher
```


## Building

```
docker build -t logsmasher .
```

## License
See LICENSE
