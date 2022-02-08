# stratumclient
[![Go Reference](https://pkg.go.dev/badge/github.com/stianwa/stratumclient.svg)](https://pkg.go.dev/github.com/stianwa/stratumclient) [![Go Report Card](https://goreportcard.com/badge/github.com/stianwa/stratumclient)](https://goreportcard.com/report/github.com/stianwa/stratumclient)

Package stratumclient implements a thin client library for the Stratum
API. The Stratum API is a query based REST API and the API itself has
no knowledge of the resources the application provides. The documentation
can usually be found on the API server at https://example.com/stratum/docs/

Installation
------------

The recommended way to install stratumclient

```
go get github.com/stianwa/stratumclient
```

Examples
--------

```go

package main

import (
        "fmt"
        "log"
        "os"
        "github.com/stianwa/stratumclient"
)

type Platform struct {
        Id       int    `json:"id"`
        Name     string `json:"name"`
        GuestOs  string `json:"guestos"`
}

func (p *Platform) String() string {
        return fmt.Sprintf("[%d] %s (%s)", p.Id, p.Name, p.GuestOs)
}

func main() {
        c := &stratumclient.Client{
                Username: "apiclienttest",
                Password: os.Getenv("STRATUM_PASSWORD"),
                BaseUrl:  "https://example.com/stratum/v1",
        }

	// Login
        if err := c.Open(); err != nil {
                log.Fatal(err)
        }

	// Add a new platform in Systembasen
        var new []*Platform
        if err := c.Post("platform/?returning=*", map[string]string{"name":"Linux new platform"}, &new); err != nil {                                                                   
                log.Fatal(err)
        }
        if len(new) != 1 {
                log.Fatal(fmt.Errorf("Failed to create a new platform, didn't return 1 row"))
        }

	// Change a platform in Systembasen
        if err := c.Put(fmt.Sprintf("platform/?where=id=%d", new[0].Id), map[string]string{"guestos":"LINUX_64"}, nil); err != nil {                                                    
                log.Fatal(err)
        }

	// List platforms matching Linux in Systembasen
        var platforms []*Platform
        if err := c.Get("platform/?where=name~Linux&select=*&orderby=name", &platforms); err != nil {
                log.Fatal(err)
        }

        for _, platform := range platforms {
                fmt.Println(platform)
        }

	// Delete a platform in Systembasen
        if err := c.Delete(fmt.Sprintf("platform/?where=id=%d", new[0].Id), nil, nil); err != nil  {                                                                                    
                log.Fatal(err)
        }

}
```

State
-------
The stratumclient module is currently under development.
