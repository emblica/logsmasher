
var console = {
  log: function(a) {
    print(JSON.stringify(a))
  }
}


function startsWith(searchString) {
  return function (p){
      if (!p) return false
      return p.substr(0, searchString.length) === searchString;
  }
}


function add_tag(message, tag) {
  return _.union(message.tags, [tag])
}

function add_tags(message, tags) {
  return _.union(message.tags, tags)
}


function run_pipeline(message) {

  var msg = steps.reduce(function(msg, step) {
    if (!msg) return msg
    return step(msg)
  }, message)
  if (msg) {
    return {
      tags: msg.tags,
      ts: msg['@timestamp'],
      msg: JSON.stringify(msg)
    }
  }
  return null
}


function path(pathArr) {
  return function(nestedObj) {
    return pathArr.reduce(function (obj, key){
        return (obj && obj[key] !== 'undefined') ? obj[key] : undefined
    }, nestedObj);
  }
}
