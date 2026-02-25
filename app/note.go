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
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

type Note struct {
	Content        string `json:"content" redis:"content"`
	IV             string `json:"iv" redis:"iv"`
	Salt           string `json:"salt" redis:"salt"`
	AllowedIPRange string `json:"allowed_ips" redis:"allowed_ips"`
	LimitClicks    bool   `json:"limit_clicks" redis:"limit_clicks"`
	MaxClicks      int    `json:"max_clicks" redis:"max_clicks"`
	CountedClicks  int    `json:"counted_clicks" redis:"counted_clicks"`
	VerifyHash     string `json:"-" redis:"verify_hash"`
	VerifySalt     string `json:"-" redis:"verify_salt"`
}

func make_ctx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 5*time.Second)
}

func (n *Note) SaveNote(id string, expiry time.Duration) error {
	ctx, cancel := make_ctx()
	defer cancel()

	err := rc.HSet(ctx, id, n).Err()
	if err != nil {
		return err
	}

	if expiry > 0 {
		ctx2, cancel2 := make_ctx()
		defer cancel2()
		err = rc.Expire(ctx2, id, expiry).Err()
		if err != nil {
			return err
		}
	}

	return nil
}

func GetNote(id string) (Note, error) {
	note_obj := Note{}

	ctx, cancel := make_ctx()
	defer cancel()

	output := rc.HGetAll(ctx, id)

	err := output.Err()
	if err != nil {
		log.Printf("Failed to get note: %s", err.Error())
		return Note{}, fmt.Errorf("Note not found")
	}

	err = output.Scan(&note_obj)
	if err != nil {
		log.Printf("Failed to scan note: %s", err.Error())
		return Note{}, fmt.Errorf("Failed to scan note")
	}

	if len(note_obj.Content) == 0 {
		return Note{}, fmt.Errorf("Note not found")
	}

	return note_obj, nil
}

// Lua script that atomically reads a note, increments click count if applicable,
// and deletes the note if max clicks is reached. This prevents the TOCTOU race
// condition where concurrent requests could read a note more times than max_clicks.
var getAndCountScript = redis.NewScript(`
local key = KEYS[1]
local data = redis.call('HGETALL', key)
if #data == 0 then
    return nil
end

local limit_clicks = false
local max_clicks = 0
for i = 1, #data, 2 do
    if data[i] == 'limit_clicks' then
        limit_clicks = (data[i+1] == '1')
    elseif data[i] == 'max_clicks' then
        max_clicks = tonumber(data[i+1])
    end
end

if limit_clicks then
    local new_count = redis.call('HINCRBY', key, 'counted_clicks', 1)
    for i = 1, #data, 2 do
        if data[i] == 'counted_clicks' then
            data[i+1] = tostring(new_count)
            break
        end
    end
    if new_count >= max_clicks then
        redis.call('DEL', key)
    end
end

return data
`)

func GetNoteAtomic(id string) (Note, error) {
	ctx, cancel := make_ctx()
	defer cancel()

	result, err := getAndCountScript.Run(ctx, rc, []string{id}).StringSlice()
	if err != nil {
		if err == redis.Nil {
			return Note{}, fmt.Errorf("Note not found")
		}
		log.Printf("Failed to get note atomically: %s", err.Error())
		return Note{}, fmt.Errorf("Note not found")
	}

	if len(result) == 0 {
		return Note{}, fmt.Errorf("Note not found")
	}

	m := make(map[string]string)
	for i := 0; i < len(result)-1; i += 2 {
		m[result[i]] = result[i+1]
	}

	note := Note{}
	note.Content = m["content"]
	note.IV = m["iv"]
	note.Salt = m["salt"]
	note.AllowedIPRange = m["allowed_ips"]
	note.LimitClicks = m["limit_clicks"] == "1"
	note.MaxClicks, _ = strconv.Atoi(m["max_clicks"])
	note.CountedClicks, _ = strconv.Atoi(m["counted_clicks"])
	note.VerifyHash = m["verify_hash"]
	note.VerifySalt = m["verify_salt"]

	if len(note.Content) == 0 {
		return Note{}, fmt.Errorf("Note not found")
	}

	return note, nil
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

func NoteExists(id string) (bool, string, string, error) {
	ctx, cancel := make_ctx()
	defer cancel()

	exists, err := rc.Exists(ctx, id).Result()
	if err != nil {
		return false, "", "", err
	}
	if exists == 0 {
		return false, "", "", nil
	}

	ctx2, cancel2 := make_ctx()
	defer cancel2()

	vhash, _ := rc.HGet(ctx2, id, "verify_hash").Result()

	ctx3, cancel3 := make_ctx()
	defer cancel3()

	vsalt, _ := rc.HGet(ctx3, id, "verify_salt").Result()

	return true, vhash, vsalt, nil
}
