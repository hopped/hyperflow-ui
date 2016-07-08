package controllers

import "github.com/revel/revel"
//import "github.com/revel/modules/jobs/app/jobs"

func init() {
//    revel.OnAppStart(func() {
//        jobs.Schedule("@every 5s", UpdateExperimentStatus{})
//    })
    revel.InterceptMethod(App.CreateExperimentsTable, revel.AFTER)
    //revel.InterceptMethod(App.UpdateExperimentStatus, revel.AFTER)
}
