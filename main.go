package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/coreos/alb-ingress-controller/controller"
	"github.com/coreos/alb-ingress-controller/controller/config"
	"github.com/coreos/alb-ingress-controller/log"
	ingresscontroller "k8s.io/ingress/core/pkg/ingress/controller"
)

func main() {
	flag.Set("logtostderr", "true")
	flag.CommandLine.Parse([]string{})

	logLevel := os.Getenv("LOG_LEVEL")
	log.SetLogLevel(logLevel)

	awsDebug, _ := strconv.ParseBool(os.Getenv("AWS_DEBUG"))

	disableRoute53, _ := strconv.ParseBool(os.Getenv("DISABLE_ROUTE53"))

	conf := &config.Config{
		ClusterName:    os.Getenv("CLUSTER_NAME"),
		AWSDebug:       awsDebug,
		DisableRoute53: disableRoute53,
	}

	port := "8080"
	http.Handle("/metrics", promhttp.Handler())
	go http.ListenAndServe(fmt.Sprintf(":%s", port), nil)

	ac := controller.NewALBController(&aws.Config{MaxRetries: aws.Int(15)}, conf)
	ic := ingresscontroller.NewIngressController(ac)

	ac.IngressClass = ic.IngressClass()
	if ac.IngressClass != "" {
		log.Infof("Ingress class set to %s", "controller", ac.IngressClass)
	}

	http.HandleFunc("/state", ac.StateHandler)

	if *ac.ClusterName == "" {
		log.Exitf("A cluster name must be defined", "controller")
	}

	if len(*ac.ClusterName) > 11 {
		log.Exitf("Cluster name must be 11 characters or less", "controller")
	}

	defer func() {
		log.Infof("Shutting down ingress controller...", "controller")
		ic.Stop()
	}()

	ac.AssembleIngresses()

	ic.Start()
}
