/*********************************
 *  File     : web.go
 *  Purpose  : Backend web logic for API endpoints
 *  Authors  : Eric Caverly
 */

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Resp struct {
	Success bool   `json:"success"`
	Data    any    `json:"data"`
	Message string `json:"message"`
}

type PostRequest struct {
	content           string
	allowed_ips       string
	days_until_expire int
	limit_clicks      bool
	max_clicks        int
	num_links         int
}

func write_error(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	bdy := Resp{
		Success: false,
		Message: msg,
		Data:    nil,
	}

	d, _ := json.Marshal(bdy)

	w.Write(d)
}

func write_success(w http.ResponseWriter, msg string, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	bdy := Resp{
		Success: true,
		Message: msg,
		Data:    data,
	}

	d_buff, err := json.Marshal(bdy)
	if err != nil {
		write_error(w, "Failed to marshal data")
		log.Printf("Failed to format a success body: %s\n", err.Error())
		return
	}

	w.Write(d_buff)
}

func web_get_note(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	log.Printf("get_note called from %s on uuid:%s\n", r.RemoteAddr, id)

	// If using a reverse proxy, which is assumed... Otherwise use r.RemoteAddr
	remote_addr := r.Header.Get("X-Real-Ip")
	if len(remote_addr) < 8 {
		remote_addr = strings.Split(r.RemoteAddr, ":")[0]
	}
	log.Printf("IP from Traefik: %s\n", remote_addr)

	n, err := GetNote(id)
	if err != nil {
		write_error(w, err.Error())
		return
	}

	allowed, err := n.CheckAllowedIP(remote_addr)
	if err != nil {
		write_error(w, "Failed to verify if your IP is allowed")
		return
	}
	if !allowed {
		write_error(w, "You are not allowed to see this note (IP restricted)")
		return
	}

	if n.LimitClicks {
		err = n.CountClicks(id)
		if err != nil {
			write_error(w, err.Error())
			return
		}
	}

	write_success(w, "Found note", n)
}

func web_post_note(w http.ResponseWriter, r *http.Request) {
	log.Printf("post_note called from %s\n", r.RemoteAddr)

	pr, err := parse_post_form(r)
	if err != nil {
		write_error(w, err.Error())
		return
	}

	id_list := []string{}

	for i := 0; i < pr.num_links; i++ {
		note := NewNote(pr.content, pr.allowed_ips, pr.limit_clicks, pr.max_clicks)
		id := uuid.NewString()
		err = note.SaveNote(id, time.Duration(pr.days_until_expire)*24*time.Hour)
		if err != nil {
			write_error(w, err.Error())
			return
		}
		id_list = append(id_list, id)
	}

	write_success(w, "Note created!", id_list)
}

func parse_post_form(r *http.Request) (PostRequest, error) {
	pr := PostRequest{}

	err := r.ParseForm()
	if err != nil {
		return pr, err
	}

	content, ok := r.Form["content"]
	if !ok {
		return pr, fmt.Errorf("Missing content in request")
	}

	if len(content[0]) > max_note_size_bytes {
		return pr, fmt.Errorf("Note too large! Max size: %d bytes", max_note_size_bytes)
	}

	if len(content[0]) == 0 {
		return pr, fmt.Errorf("Note content cannot be empty")
	}

	allowed_ips, ok := r.Form["allowed_ips"]
	if !ok {
		return pr, fmt.Errorf("Missing allowed_ips in request")
	}

	if err := check_valid_ranges(allowed_ips[0]); err != nil {
		return pr, fmt.Errorf("Invalid IP range (%s). Please enter as 1.1.1.0/24, 2.2.0.0/16", err.Error())
	}

	dte, ok := r.Form["days_until_expire"]
	if !ok {
		return pr, fmt.Errorf("Missing days_until_expire in request")
	}

	ndays, err := strconv.Atoi(dte[0])
	if err != nil || ndays < 0 || ndays > max_days {
		return pr, fmt.Errorf("Invalid number of days")
	}

	limit_clicks, ok := r.Form["limit_clicks"]
	if !ok {
		return pr, fmt.Errorf("Missing whether or not to limit clicks in request")
	}

	max_clicks_s, ok := r.Form["max_clicks"]
	if !ok && (limit_clicks[0] == "true") {
		return pr, fmt.Errorf("Missing max clicks in request")
	}

	max_clicks, err := strconv.Atoi(max_clicks_s[0])
	if err != nil {
		return pr, fmt.Errorf("Invalid max clicks value, must be int")
	}

	num_links_s, ok := r.Form["num_links"]
	if !ok {
		return pr, fmt.Errorf("Missing num links in request")
	}
	num_links, err := strconv.Atoi(num_links_s[0])
	if err != nil {
		return pr, fmt.Errorf("Invalid num links, must be int")
	}

	return PostRequest{
		content:           content[0],
		allowed_ips:       allowed_ips[0],
		days_until_expire: ndays,
		limit_clicks:      limit_clicks[0] == "true",
		max_clicks:        max_clicks,
		num_links:         num_links,
	}, nil
}

func web_health_check(w http.ResponseWriter, _ *http.Request) {
	write_success(w, "Alive", nil)
}
