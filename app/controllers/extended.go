package controllers

import (
    "github.com/revel/revel"
    "net/http"
    "bytes"
)

type Extended struct {
    *revel.Controller
}

func (c Extended) Index() revel.Result {
    c.RenderArgs["workflow"] = "Please add your HyperFlow workflow description here..."
    return c.Render()
}

func (c Extended) NewWorkflow(workflow string) revel.Result {
    c.Validation.Required(workflow).Message("Workflow description is required!")

    if workflow == "Please add your HyperFlow workflow description here..." {
        return c.RenderTemplate("Extended/Index.html")
    }

    // error pop-ups
    if c.Validation.HasErrors() {
        c.Validation.Keep()
        c.FlashParams()
        return c.Redirect(Extended.Index)
    }

    req, err := http.NewRequest("POST", defaultBaseUrl, bytes.NewBuffer([]byte(workflow)))
    if err != nil {
        panic(err)
    }
    req.Header.Set("Content-Type", mediaType)
    req.Header.Set("Connection", "close")

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        panic(err)
    }
    defer resp.Body.Close()

    // status CREATED is expected
    if resp.StatusCode != http.StatusCreated {
        c.RenderArgs["workflow"] = resp
    } else {
        c.RenderArgs["workflow"] = "Job submitted!"
    }

    return c.RenderTemplate("Extended/Index.html")
}
