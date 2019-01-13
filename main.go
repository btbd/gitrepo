package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
)

const API = "https://api.github.com"

type Repository struct {
	Name              string `json:"name,omitempty"`
	Description       string `json:"description,omitempty"`
	Homepage          string `json:"homepage,omitempty"`
	Private           bool   `json:"private,omitempty"`
	HasIssues         bool   `json:"has_issues,omitempty"`
	HasProjects       bool   `json:"has_projects,omitempty"`
	HasWiki           bool   `json:"has_wiki,omitempty"`
	TeamId            int    `json:"team_id,omitempty"`
	AutoInit          bool   `json:"auto_init,omitempty"`
	GitignoreTemplate string `json:"gitignore_template,omitempty"`
	LicenseTemplate   string `json:"license_template,omitempty"`
	AllowSquashMerge  bool   `json:"allow_squash_merge,omitempty"`
	AllowMergeCommit  bool   `json:"allow_merge_commit,omitempty"`
	AllowRebaseMerge  bool   `json:"allow_rebase_merge,omitempty"`
	Archived          bool   `json:"archived"`
}

func Request(auth string, method string, url string, body io.Reader) (int, []byte, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return 0, nil, err
	}

	req.Header.Set("Authorization", "Basic "+auth)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, err
	}

	defer resp.Body.Close()

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, err
	}

	return resp.StatusCode, b, nil
}

func QRequest(auth string, method string, url string, data io.Reader) error {
	status, body, err := Request(auth, method, url, data)
	if err != nil {
		return err
	}

	if status/100 != 2 {
		return errors.New(string(body))
	}

	return nil
}

func CreateRepository(auth string, org string, repo Repository) error {
	data, err := json.Marshal(&repo)
	if err != nil {
		return err
	}

	return QRequest(auth, "POST", API+"/orgs/"+org+"/repos", bytes.NewBuffer([]byte(data)))
}

func EditRepository(auth string, org string, name string, repo Repository) error {
	data, err := json.Marshal(&repo)
	if err != nil {
		return err
	}

	return QRequest(auth, "PATCH", API+"/repos/"+org+"/"+name, bytes.NewBuffer([]byte(data)))
}

func AddCollaborator(auth string, org string, repo string, user string) error {
	data := `{"permission":"admin"}`
	return QRequest(auth, "PUT", API+"/repos/"+org+"/"+repo+"/collaborators/"+user, bytes.NewBuffer([]byte(data)))
}

func RemoveCollaborator(auth string, org string, repo string, user string) error {
	return QRequest(auth, "DELETE", API+"/repos/"+org+"/"+repo+"/collaborators/"+user, nil)
}

func AddMember(auth string, org string, user string) error {
	data := `{"role":"member"}`
	return QRequest(auth, "PUT", API+"/orgs/"+org+"/memberships/"+user, bytes.NewBuffer([]byte(data)))
}

func RemoveMember(auth string, org string, user string) error {
	return QRequest(auth, "DELETE", API+"/orgs/"+org+"/memberships/"+user, nil)
}

func main() {
	repo := Repository{
		LicenseTemplate: "apache-2.0",
		Private:         false,
		HasIssues:       true,
	}

	token := ""
	create := false
	org := false
	add := ""
	rm := ""

	flag.StringVar(&token, "t", token, "user API token (required)")
	flag.BoolVar(&create, "c", create, "create")
	flag.BoolVar(&org, "o", org, "add members to org")
	flag.StringVar(&add, "a", add, "add admin members")
	flag.StringVar(&rm, "r", rm, "remove admin members")
	flag.StringVar(&repo.Description, "d", repo.Description, "description")

	flag.Parse()

	if token == "" {
		flag.Usage()
		os.Exit(0)
	}

	auth := base64.StdEncoding.EncodeToString([]byte(":" + token))

	var repos [][]string
	for _, a := range flag.Args() {
		s := strings.Split(a, "/")
		if len(s) != 2 {
			fmt.Fprintln(os.Stderr, "repos must be in format ORG/REPO")
			os.Exit(1)
		}

		repos = append(repos, s)
	}

	if len(repos) == 0 {
		fmt.Fprintln(os.Stderr, "no repos specified")
		os.Exit(1)
	}

	var add_users []string
	if add != "" {
		for _, u := range strings.Split(add, ",") {
			if err := QRequest(auth, "GET", API+"/users/"+u, nil); err != nil {
				fmt.Fprintf(os.Stderr, "failed to get user: %v\n", err)
				os.Exit(1)
			}

			add_users = append(add_users, u)
		}
	}

	var rm_users []string
	if rm != "" {
		for _, u := range strings.Split(rm, ",") {
			if err := QRequest(auth, "GET", API+"/users/"+u, nil); err != nil {
				fmt.Fprintf(os.Stderr, "failed to get user: %v\n", err)
				os.Exit(1)
			}

			rm_users = append(rm_users, u)
		}
	}

	for _, r := range repos {
		repo.Name = r[1]
		name := r[0] + "/" + r[1]

		if create {
			if err := CreateRepository(auth, r[0], repo); err != nil {
				fmt.Fprintf(os.Stderr, "failed to create repo %q: %v\n", name, err)
				os.Exit(1)
			}

			fmt.Printf("%q created\n", name)
		} else {
			if err := EditRepository(auth, r[0], r[1], repo); err != nil {
				fmt.Fprintf(os.Stderr, "failed to edit repo %q: %v\n", name, err)
				os.Exit(1)
			}

			fmt.Printf("%q edited\n", name)
		}

		for _, u := range add_users {
			if err := AddCollaborator(auth, r[0], r[1], u); err != nil {
				fmt.Fprintf(os.Stderr, "failed to add user %q to %q: %v\n", u, name, err)
				os.Exit(1)
			}

			fmt.Printf("- added %q to %q\n", u, name)

			if org {
				if err := QRequest(auth, "GET", API+"/orgs/"+r[0]+"/members/"+u, nil); err == nil {
					fmt.Printf("- %q is already a member of %q\n", u, r[0])
				} else {
					if err := AddMember(auth, r[0], u); err != nil {
						fmt.Fprintf(os.Stderr, "failed to add member %q to org %q: %v\n", u, r[0], err)
						os.Exit(1)
					}

					fmt.Printf("- added %q to %q\n", u, r[0])
				}
			}
		}

		for _, u := range rm_users {
			if err := RemoveCollaborator(auth, r[0], r[1], u); err != nil {
				fmt.Fprintf(os.Stderr, "failed to remove user %q to %q: %v\n", u, name, err)
				os.Exit(1)
			}

			fmt.Printf("- removed %q from %q\n", u, name)
		}
	}
}
