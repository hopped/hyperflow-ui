package controllers

import (
    "github.com/revel/revel"
)

type About struct {
    *revel.Controller
}

func (c About) Index() revel.Result {
    return c.Render()
}
