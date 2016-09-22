package controllers

import (
    "github.com/revel/revel"
)

type Contact struct {
    *revel.Controller
}

func (c Contact) Index() revel.Result {
    return c.Render()
}
