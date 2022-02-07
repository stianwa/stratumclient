package stratumclient

import (
        "testing"
	"fmt"
	"os"
)

var platformId int

var c *Client = &Client{
	Username: "apiclienttest",
	Password: os.Getenv("STRATUM_PASSWORD"),
	BaseUrl:  "https://" + os.Getenv("STRATUM_HOST") + "/stratum/v1",
}

type Platform struct {
	Id       int    `json:"id"`
	Name     string `json:"name"`
	GuestOs  string `json:"guestos"`
	ImageUrl string `json:"image_url"`
}

func (p *Platform) String() string {
	return fmt.Sprintf("[%d] %s", p.Id, p.Name)
}

func TestOpen(t *testing.T) {
	if c.Password == "" {
		fmt.Println("API-password must be provided in the STRATUM_PASSWORD environment variable!")
		os.Exit(1)
	}

	if err := c.Open(); err != nil {
                t.Fatalf("open: %v\n", err)
        }
}

func TestGet(t *testing.T) {
	var p []*Platform
	if err := c.Get("platform/?orderby=name&select=id,name&where=name~Linux", &p); err != nil {
                t.Fatalf("get platforms: %v\n", err)
	}

	if len(p) < 4 {
                t.Fatalf("get platforms count: %d", len(p))
	}
}

func TestPost(t *testing.T) {
	post := make(map[string]string)
	post["name"] = "Linux SuperCoreFlashyPlatform 1.0"

	var p []*Platform
	if err := c.Post("platform/?returning=*", post, &p); err != nil {
                t.Fatalf("post platform: %v\n", err)
	}

	if len(p) != 1 {
                t.Fatalf("get platform count: %d", len(p))
	}
	platformId = p[0].Id
}

func TestPut(t *testing.T) {
	post := make(map[string]string)
	post["guestos"] = "NOSUCHTHING"

	var p []*Platform
	if err := c.Put("platform/?returning=*&where=id=" + fmt.Sprintf("%d", platformId), post, &p); err != nil {
                t.Fatalf("put platform: %v\n", err)
	}

	if len(p) != 1 {
                t.Fatalf("put platform count: %d", len(p))
	}
}

func TestDelete(t *testing.T) {
	var p []*Platform
	if err := c.Delete("platform/?returning=*&where=id=" + fmt.Sprintf("%d", platformId), nil, &p); err != nil {
                t.Fatalf("delete platform: %v\n", err)
	}

	if len(p) != 1 {
                t.Fatalf("delete platform count: %d", len(p))
	}
}
