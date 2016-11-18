package controllers

import (
    "github.com/revel/revel"
    "net/http"
)

type About struct {
    *revel.Controller
}

func (c About) Index() revel.Result {
    return c.Render()
}

type Message struct {
    Status string `json:"status"`
}

func (c About) UpdateExperiment(s3Resource string) revel.Result {
    response, err := http.Head(s3Resource)
    if err != nil {
        return c.RenderJson(Message{Status: "Running"})
    } else if response.StatusCode == 200 {
        return c.RenderJson(Message{Status: "Finished"})
    }
    return c.RenderJson(Message{Status: "Running"})
}
