/*********************************
 *  File     : note.go
 *  Purpose  : Defines what a note is
 *  Authors  : Eric Caverly
 */

package main

import (
	"context"
	"fmt"
	"log"
	"time"
)

type Note struct {
	Content        string `json:"content" redis:"content"`
	IV             string `json:"iv" redis:"iv"`
	Salt           string `json:"salt" redis:"salt"`
	AllowedIPRange string `json:"allowed_ips" redis:"allowed_ips"`
	LimitClicks    bool   `json:"limit_clicks" redis:"limit_clicks"`
	MaxClicks      int    `json:"max_clicks" redis:"max_clicks"`
	CountedClicks  int    `json:"counted_clicks" redis:"counted_clicks"`
}

func make_ctx() context.Context {
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	return ctx
}

func (n *Note) SaveNote(id string, expiry time.Duration) error {
	err := rc.HSet(make_ctx(), id, n).Err()
	if err != nil {
		return err
	}

	err = rc.Expire(make_ctx(), id, expiry).Err()
	if err != nil {
		return err
	}

	return nil
}

func GetNote(id string) (Note, error) {
	note_obj := Note{}

	output := rc.HGetAll(make_ctx(), id)
	log.Printf("%#v\n", output)

	err := output.Err()
	if err != nil {
		log.Printf("Failed to get note: %s", err.Error())
		return Note{}, fmt.Errorf("Note not found")
	}

	err = output.Scan(&note_obj)
	if err != nil {
		log.Printf("Failed to scan note: %s", err.Error())
		return Note{}, fmt.Errorf("Failed to scan node")
	}

	if len(note_obj.Content) == 0 {
		return Note{}, fmt.Errorf("Note not found")
	}

	return note_obj, nil
}

func (n *Note) CheckAllowedIP(source_ip string) (bool, error) {
	allowed, err := within_ranges(source_ip, n.AllowedIPRange)
	if err != nil {
		return false, fmt.Errorf("Failed to check if source IP was valid: %s", err.Error())
	}

	if !allowed {
		return false, fmt.Errorf("You are not allowed to access this note! (IP address forbidden)")
	}

	return true, nil
}

func (n *Note) CountClicks(id string) error {
	result := rc.HIncrBy(make_ctx(), id, "counted_clicks", 1)
	if result.Err() != nil {
		return result.Err()
	}

	n.CountedClicks = int(result.Val())

	if n.CountedClicks == n.MaxClicks {
		err := rc.Del(make_ctx(), id).Err()
		if err != nil {
			return err
		}
		log.Printf("Note %s has exceeded max clicks, removing", id)
	} else {
		log.Printf("Note %s is now at %d/%d clicks", id, n.CountedClicks, n.MaxClicks)
	}

	return nil
}
