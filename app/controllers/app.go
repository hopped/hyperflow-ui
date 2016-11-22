package controllers

import (
    "bytes"
    "encoding/json"
    "github.com/revel/revel"
    "github.com/revel/revel/cache"
    "strings"
    "strconv"
    "text/template"
    "time"
    "net/http"
)

const (
    defaultBaseUrl = "http://localhost:51404/apps"
    mediaType = "application/json"
)

type Config struct {
    BaseURL string
}

type App struct {
	*revel.Controller
}

func (c App) Index() revel.Result {
    c.RenderArgs["molecules"] = strconv.FormatUint(uint64(1000), 10)
    c.RenderArgs["temperature"] = strconv.FormatInt(int64(85), 10)
    c.RenderArgs["simulation_end_time"] = strconv.FormatFloat(float64(5.5), 'f', -1, 64)
	return c.Render()
}

type WorkflowObject struct {
    Processes   []ProcessesType `json:"processes"`
    Signals     []SignalsType `json:"signals"`
    Ins         []string `json:"ins"`
    Outs        []string `json:"outs"`
}

type ProcessesType struct {
    Name        string `json:"name"`
    Function    string `json:"function"`
    Type        string `json:"type"`
    Config      ConfigType `json:"config"`
    Limit       int `json:"firingLimit"`
    Ins         []string `json:"ins"`
    Outs        []string `json:"outs"`
}

type SignalsType struct {
    Name        string `json:"name"`
    Data        []string `json:"data,omitempty"`
}

type ConfigType struct {
    Executor    ExecutorType `json:"executor"`
}

type ExecutorType struct {
    Executable  string `json:"executable"`
    Args        []string `json:"args"`
}

type Filenames struct {
    FilenameOutArchived string
    FilenameOutVideo string
}

type Experiment struct {
    Stamp           string
    Molecules       string
    Temperature     string
    SimulationTime  string
    Status          string
    Artifacts       string
    Tgz             string
    Avi             string
    Workflow        WorkflowObject
}

func (c App) NewWorkflow(
    number_of_molecules uint,
    temperature int,
    simulation_end_time float32,
    record_movie bool) revel.Result {

    // validate incoming form data
    c.Validation.Required(number_of_molecules).Message("Number of molecules is required")
    c.Validation.Range(temperature, 0, 100).Message("Temperature is required (in Celsius)")
    c.Validation.Required(simulation_end_time).Message("End time of simulation is required (in seconds)")

    // error pop-ups
    if c.Validation.HasErrors() {
        c.Validation.Keep()
        c.FlashParams()
        return c.Redirect(App.Index)
    }

    // read workflow template and adapt filenames
    now := strconv.FormatInt(int64(time.Now().Unix()), 10)
    var defaultWorkflow bytes.Buffer
    t := template.New("workflow.json")
    t, _ = t.ParseFiles(revel.BasePath + "/conf/workflow.json")
    f := Filenames {
        FilenameOutArchived: "md-simulation-" + now + ".tgz",
        FilenameOutVideo: "md-simulation-" + now + ".avi",
    }
    err := t.Execute(&defaultWorkflow, f)
    if err != nil {
        panic(err)
    }

    // parse template into JSON object
    var workflowDescription WorkflowObject
    err = json.Unmarshal(defaultWorkflow.Bytes(), &workflowDescription)
    if (err != nil) {
        panic(err)
    }

    // modify default description as requested by user
    for i := range workflowDescription.Processes {
        if strings.HasPrefix(workflowDescription.Processes[i].Name, "run-cmd") {
            // number of molecules
            strNumOfMolecules := strconv.FormatUint(uint64(number_of_molecules), 10)
            workflowDescription.Processes[i].Config.Executor.Args[0] = strNumOfMolecules
            // simulation end time
            strSimulationEndTime := strconv.FormatFloat(float64(simulation_end_time), 'f', -1, 64)
            workflowDescription.Processes[i].Config.Executor.Args[1] = strSimulationEndTime
            // temperature
            strTemperature := strconv.FormatInt(int64(temperature), 10)
            workflowDescription.Processes[i].Config.Executor.Args[2] = strTemperature
            // append output filename
            workflowDescription.Processes[i].Config.Executor.Args = append(workflowDescription.Processes[i].Config.Executor.Args, f.FilenameOutArchived)
        }
    }

    // keep or remove movie generation
    if record_movie == false {
        // remove make-movie task
        for i := range workflowDescription.Processes {
            if strings.HasPrefix(workflowDescription.Processes[i].Name, "make-movie") {
                workflowDescription.Processes = append(workflowDescription.Processes[:i], workflowDescription.Processes[i+1:]...)
            }
        }
        // remove any signals related to AVI files
        for i := range workflowDescription.Signals {
            if strings.HasSuffix(workflowDescription.Signals[i].Name, ".avi") {
                workflowDescription.Signals = append(workflowDescription.Signals[:i], workflowDescription.Signals[i+1:]...)
            }
        }
        // remove any output strings related to AVI files
        for i := range workflowDescription.Outs {
            if strings.HasSuffix(workflowDescription.Outs[i], ".avi") {
                workflowDescription.Outs = append(workflowDescription.Outs[:i], workflowDescription.Outs[i+1:]...)
            }
        }
    }

    // post workflow to HyperFlow
    b, err := json.Marshal(workflowDescription)
    if err != nil {
        panic(err)
    }
    req, err := http.NewRequest("POST", defaultBaseUrl, bytes.NewBuffer(b))
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
    statusURL := ""
    if resp.StatusCode != http.StatusCreated {
        panic(resp)
    } else {
        location, err := resp.Location()
        if err != nil {
            panic(err)
        }
        statusURL = location.String()
    }

    timestamp := time.Now()
    timestamp_str := timestamp.Format(time.Stamp)

    c.RenderArgs["filenames"] = f
    c.RenderArgs["simulation_end_time"] = simulation_end_time
    c.RenderArgs["molecules"] = number_of_molecules
    c.RenderArgs["timestamp"] = timestamp_str
    c.RenderArgs["temperature"] = temperature
    c.RenderArgs["statusURL"] = statusURL
    c.RenderArgs["workflow"] = workflowDescription
    return c.RenderTemplate("App/Index.html")
}

func (c App) CreateExperimentsTable() revel.Result {
    var newExperiment Experiment
    var experimentIds []string
    if err := cache.Get("experiment_ids", &experimentIds); err != nil {
        // no data, empty slice
    }
    if c.RenderArgs["timestamp"] != nil {
        temperature := c.RenderArgs["temperature"].(int)
        molecules := c.RenderArgs["molecules"].(uint)
        simulation_end_time := c.RenderArgs["simulation_end_time"].(float32)

        newExperiment.Stamp = c.RenderArgs["timestamp"].(string)
        newExperiment.Temperature = strconv.FormatInt(int64(temperature), 10)
        newExperiment.Molecules = strconv.FormatInt(int64(molecules), 10)
        newExperiment.SimulationTime = strconv.FormatFloat(float64(simulation_end_time), 'f', -1, 64)
        newExperiment.Status = "Running"
        filenames := c.RenderArgs["filenames"].(Filenames)
        newExperiment.Tgz = filenames.FilenameOutArchived
        newExperiment.Avi = filenames.FilenameOutVideo
        newExperiment.Workflow = c.RenderArgs["workflow"].(WorkflowObject)

        // cache.DEFAULT == one hour persistency
        if err := cache.Add(newExperiment.Stamp, newExperiment, cache.DEFAULT); err == nil {
            // new experiment
            experimentIds = append([]string{ newExperiment.Stamp }, experimentIds...)
            cache.Set("experiment_ids", experimentIds, cache.DEFAULT)
        }
    }

    type ExperimentList []Experiment
    var myExperiments ExperimentList
    if g, err := cache.GetMulti(experimentIds...); err == nil {
        var oldExperiment Experiment
        for _, value := range experimentIds {
            if err := g.Get(value, &oldExperiment); err == nil {
                myExperiments = append(myExperiments, oldExperiment)
            }
        }
    }

    // provide S3 base url
    s3_region := revel.Config.StringDefault("aws.s3.region", "eu-central-1")
    s3_bucket := revel.Config.StringDefault("aws.s3.bucket", "paasage-md-bucket")
    s3_path := revel.Config.StringDefault("aws.s3.path", "results/")
    s3_base := "http://s3." + s3_region + ".amazonaws.com/" + s3_bucket + "/" + s3_path

    c.RenderArgs["s3_base"] = s3_base
	c.RenderArgs["myExperiments"] = myExperiments

    return c.RenderTemplate("App/Index.html")
}

