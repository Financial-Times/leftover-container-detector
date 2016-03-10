package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v2"
)

type services struct {
	Services []service `yaml:"services"`
}

type service struct {
	Name    string `yaml:"name"`
	Version string `yaml:"version"`
	Count   int    `yaml:"count"`
}

type container struct {
	Name    string
	Version string
	Count   int
	Service service
}

func main() {
	dat, _ := ioutil.ReadFile("services.yaml")
	var servicesList services
	yaml.Unmarshal(dat, &servicesList)

	for i, service := range servicesList.Services {
		if strings.Contains(service.Name, "sidekick") || strings.Contains(service.Name, "timer") {
			servicesList.Services = append(servicesList.Services[:i], servicesList.Services[i+1:]...)
		}
	}

	for i, service := range servicesList.Services {
		if strings.Contains(service.Name, "@.service") {
			servicesList.Services[i].Name = strings.Replace(service.Name, "@.service", "", 1)
		} else {
			servicesList.Services[i].Name = strings.Replace(service.Name, ".service", "", 1)
		}
	}

	singleServices := []string{"fleet-unit-healthcheck", "kafka-rest-proxy", "vulcan-config-builder", "kafka", "zookeeper"}
	generalServices := []string{"diamond", "system-healthcheck", "vulcan", "elb-presence"}

	for i, service := range servicesList.Services {
		if stringInSlice(service.Name, singleServices) {
			servicesList.Services[i].Count = 1
			continue
		}
		if stringInSlice(service.Name, generalServices) {
			servicesList.Services[i].Count = 5
			continue
		}
	}
	//fmt.Printf("Yaml: \n %# v\n", pretty.Formatter(servicesList))

	dat, _ = ioutil.ReadFile("serviceToImageName.map")
	var objmap map[string]*json.RawMessage
	json.Unmarshal(dat, &objmap)
	//fmt.Printf("Mapping: \n %# v\n", pretty.Formatter(objmap))

	imagesToIgnore := []string{"ONE-SHOT", "IGNORE", "NO-CONTAINER"}

	serviceToImage := make(map[string]string)
	imageToService := make(map[string]service)

	var servicesToIgnore []string
	for serviceName, imageNameJSON := range objmap {
		var imageName string
		json.Unmarshal(*imageNameJSON, &imageName)

		if stringInSlice(imageName, imagesToIgnore) {
			servicesToIgnore = append(servicesToIgnore, serviceName)
		} else {
			serviceToImage[serviceName] = imageName
			for _, service := range servicesList.Services {
				if service.Name == serviceName {
					imageToService[imageName] = service
					break
				}
			}
		}
	}
	imageToService["coco-fleet-deployer"] = service{Name: "deployer", Version: "latest", Count: 1}
	imageToService["coco-logfilter"] = service{Name: "splunk-forwarder", Count: 5}
	imageToService["coco-splunk-http-forwarder"] = service{Name: "splunk-forwarder", Count: 5}

	//fmt.Printf("Mapping service to image: \n %# v\n", pretty.Formatter(serviceToImage))
	//fmt.Printf("Mapping image to service: \n %# v\n", pretty.Formatter(imageToService))
	//fmt.Printf("Ignored: \n %# v\n", pretty.Formatter(servicesToIgnore))

	var containers []container
	for _, val := range strings.Split(os.Args[1], "\n") {
		parts := strings.Split(strings.TrimSpace(val), " ")
		count, _ := strconv.Atoi(parts[0])
		containerNameParts := strings.Split(parts[1], "/")
		containerNameWithVersion := containerNameParts[len(containerNameParts)-1]
		var containerName string
		version := ""
		if strings.Contains(containerNameWithVersion, ":") {
			containerNameParts = strings.Split(containerNameWithVersion, ":")
			containerName = containerNameParts[0]
			version = containerNameParts[1]
		} else {
			containerName = containerNameWithVersion
		}
		service := imageToService[containerName]
		if service.Name == "" {
			fmt.Printf("Ignoring Service: %v\n", containerName)
		}
		containers = append(containers, container{Name: containerName, Version: version, Count: count, Service: service})
	}

	//fmt.Printf("Containers: \n %# v\n", pretty.Formatter(containers))

	ignoreVersion := []string{"coco-fleet-unit-healthcheck", "neo4j", "kafka", "coco-system-healthcheck"}

	for _, container := range containers {
		countWrong := container.Count != container.Service.Count
		versionWrong := container.Version != container.Service.Version && !stringInSlice(container.Name, ignoreVersion)

		if versionWrong {
			fmt.Printf("%v: we expected version [%v] but found version [%v] with [%v] instances.\n", container.Name, container.Service.Version, container.Version, container.Count)
		} else if countWrong {
			totalNrOfExpectedInstances := 0
			for _, image := range serviceToImage {
				if image == container.Name {
					for _, service := range imageToService {
						if service.Name == container.Service.Name {
							totalNrOfExpectedInstances += service.Count
						}
					}
				}
			}
			if totalNrOfExpectedInstances == container.Count {
				fmt.Printf("%v has [%v] instances but is used by several services.\n", container.Name, totalNrOfExpectedInstances)
			} else {
				fmt.Printf("%v: we expected [%v] instances but there are [%v] instances.\n", container.Name, container.Service.Count, container.Count)
			}
		}
	}

}

func stringInSlice(str string, list []string) bool {
	for _, v := range list {
		if v == str {
			return true
		}
	}
	return false
}
