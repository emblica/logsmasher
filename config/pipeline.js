


function only_logs (message) {
  // Drop events with wrong ingest path
  // Only /_bulk is allowed
  if (message.headers.request_path !== '/_bulk') {
    return null
  }
  // Drop events without 'tag'-field
  // Prevents other than Fluentd-events because all of them got this
  if (!message.tag) {
    return null
  }
  message.tags = [];
  return message
}



// Remove headers and fluenttag-property
function strip_headers (message) {
  return _.chain(message)
    .omit('fluenttag')
    .omit('headers')
    .value()
}

// Trim whitespace and newlines from logs
function trim (message) {
  if (message.log) {
    message.log = message.log.trim()
  }
  return message
}


// Container STDOUTs
// The same you can see with
// kubectl logs ...
function kubernetes (message) {
  if (startsWith("kubernetes.var.log.containers")(message.tag)) {
    message.tags = add_tag(message, "container-log")
  }
  return message
}

var container_name = path(['kubernetes', 'container_name'])
var namespace_name = path(['kubernetes', 'namespace_name'])

// # Calico is an overlay network for kubernetes
// # This tags it with 'networking'
function calico (message) {
  if (container_name(message)=== "calico-node") {
    //message.body = parsed
    message.tags = add_tag(message, "networking")
  }
  return message
}

// This parses fluentd own logs from DaemonSet
function fluentd (message) {
  if (container_name(message) === "fluentd") {
      message.tags = add_tag(message, "logging")
  }
  return message
}


// Mark everything from kube-system as admin
function kube_system_to_admin (message) {
  if (namespace_name(message) === "kube-system") {
      message.tags = add_tag(message, "admin")
  }
  return message
}

// Mark everything from team_1 namespace
function team_1_tags (message) {
  if (startsWith("team_1")(namespace_name(message))) {
      message.tags = add_tag(message, "team-1")
      // Mark messages from gdpr-container as confidential
      if (container_name(message) === "gdpr") {
          message.tags = add_tag(message, "confidential")
      }
  }
  return message
}


// Mark everything from team_1 namespace
function team_2_tags (message) {
  if (startsWith("team_2")(namespace_name(message))) {
      message.tags = add_tag(message, "team-2")
      // Mark messages from gdpr-container as confidential
      if (container_name(message) === "gdpr") {
          message.tags = add_tag(message, "confidential")
      }
  }
  return message
}




steps = [
  only_logs,
  strip_headers,
  trim,
  kubernetes,
  calico,
  fluentd,
  kube_system_to_admin,
  team_1_tags,
  team_2_tags
]
