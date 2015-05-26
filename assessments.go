package main


import (
  "time"
  "errors"
  "net"
  "net/rpc"
  "net/http"
  "github.com/onlineta/common"
)

type subscription struct {
  id string
  ch chan string
}

var request_ch chan string
var subscribe_ch chan subscription
var delete_ch chan string
var result_ch chan error
var incoming_ch chan Assessment

type Receiver int

// Used by clients to send an assessment
type Assessment struct {
  id string
  assess string
}

type Assessments struct {
  subscriptions map[string]chan string
}


func New() Assessments {
  assess := Assessments{}
  return assess
}

func (a *Assessments) Serve() {
  subscribe_ch = make(chan subscription)
  request_ch = make(chan string)
  result_ch = make(chan error)
  delete_ch = make(chan string)
  incoming_ch = make(chan Assessment)

  a.subscriptions = make(map[string]chan string)
  go func() {
    for {
      select {
      case subs := <- subscribe_ch:
        if _, ok := a.subscriptions[subs.id]; ok {
          a.subscriptions[subs.id] = subs.ch
        } else {
          result_ch <- errors.New("foo")
        }
      case req := <- request_ch:
        // TODO: lots of stuff...
        // Issue a new request for receiving an assessment
        _ = req
      case in := <- incoming_ch:
        // If the id of the incoming submission belongs to a subscribed
        // submission, forward that assessment to the subscribed submission
        // Otherwise, simply throw it away
        if ch, ok := a.subscriptions[in.id]; ok  {
          ch <- in.assess
          delete(a.subscriptions, in.id)
        }
      case del := <- delete_ch:
        if _, err := a.subscriptions[del]; err {
          delete(a.subscriptions, del)
        }
      }
    }
  }()

  // Start RPC server
  receiver := new(Receiver)
  rpc.Register(receiver)
  rpc.HandleHTTP()
  l, err := net.Listen("tcp", common.ConfigValue("AssessmentPort"))
  if err != nil {
    panic("Couldn't start listener")
  }
  go http.Serve(l, nil)
}


func store_if_not_exists(id string, ch chan string) error {
  subscribe_ch <- subscription{
    id: id,
    ch: ch,
  }
  if result := <- result_ch; result != nil {
    return errors.New("Already exists")
  }
  return nil
}

// Called by a submission to subscribe to an assessment
// When the assessment is ready, it will be sent over the
// channel returned by this function
func Subscribe(id string) (<- chan string, bool) {
  proxy_ch := make(chan string)
  ch := make(chan string)
  if err := store_if_not_exists(id, proxy_ch); err != nil {
    return nil, true
  }

  go func() {
    select {
    case text := <- proxy_ch:
      ch <- text
    case <- time.After(time.Duration(common.ConfigIntValue("AssessmentTimeout") * 1e9)):
      ch <- ""
    }
  }()
  return ch, false
}

//func (r *Receiver) Send(assess *Assessment, reply *int) {
