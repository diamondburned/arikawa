// Package infer implements reflect functions that package bot uses.
//
// Functions in this package may run recursively forever. This shouldn't happen
// with Arikawa's structures, but use these functions with care.
package infer

import (
	"reflect"
	"strings"

	"github.com/diamondburned/arikawa/discord"
)

// ChannelID looks for fields with name ChannelID, Channel, or in some special
// cases, ID.
func ChannelID(event interface{}) discord.Snowflake {
	return reflectID(reflect.ValueOf(event), "Channel")
}

// GuildID looks for fields with name GuildID, Guild, or in some special cases,
// ID.
func GuildID(event interface{}) discord.Snowflake {
	return reflectID(reflect.ValueOf(event), "Guild")
}

// UserID looks for fields with name UserID, User, or in some special cases, ID.
func UserID(event interface{}) discord.Snowflake {
	// This may have a very fatal bug of accidentally mistaking another User's
	// ID. It also probably wouldn't work with things like RecipientID.
	return reflectID(reflect.ValueOf(event), "User")
}

func reflectID(v reflect.Value, thing string) discord.Snowflake {
	if !v.IsValid() {
		return 0
	}

	t := v.Type()

	if t.Kind() == reflect.Ptr {
		v = v.Elem()

		// Recheck after dereferring
		if !v.IsValid() {
			return 0
		}

		t = v.Type()
	}

	if t.Kind() != reflect.Struct {
		return 0
	}

	numFields := t.NumField()

	for i := 0; i < numFields; i++ {
		field := t.Field(i)
		fType := field.Type

		if fType.Kind() == reflect.Ptr {
			fType = fType.Elem()
		}

		switch fType.Kind() {
		case reflect.Struct:
			if chID := reflectID(v.Field(i), thing); chID.Valid() {
				return chID
			}
		case reflect.Int64:
			switch {
			case false,
				// Contains works with "LastMessageID" and such.
				strings.Contains(field.Name, thing+"ID"),
				// Special case where the struct name has Channel in it.
				field.Name == "ID" && strings.Contains(t.Name(), thing):

				return discord.Snowflake(v.Field(i).Int())
			}
		}
	}

	return 0
}

/*
var reflectCache sync.Map

type cacheKey struct {
	t reflect.Type
	f string
}

func getID(v reflect.Value, thing string) discord.Snowflake {
	if !v.IsValid() {
		return 0
	}

	t := v.Type()

	if t.Kind() == reflect.Ptr {
		v = v.Elem()

		// Recheck after dereferring
		if !v.IsValid() {
			return 0
		}

		t = v.Type()
	}

	if t.Kind() != reflect.Struct {
		return 0
	}

	return reflectID(thing, v, t)
}

type reflector struct {
	steps   []step
	thing   string
	thingID string
}

type step struct {
	field int
	ptr   bool
	rec   []step
}

func reflectID(thing string, v reflect.Value, t reflect.Type) discord.Snowflake {
	r := &reflector{thing: thing}

	// copy original type
	key := r.thing + t.String()

	// check the cache
	if instructions, ok := reflectCache.Load(key); ok {
		if instructions == nil {
			return 0
		}
		return applyInstructions(v, instructions.([]step))
	}

	r.thingID = r.thing + "ID"
	r.steps = make([]step, 0, 1)
	id := r._id(v, t)

	if r.steps != nil {
		reflectCache.Store(key, r.instructions())
	}

	return id
}

func applyInstructions(v reflect.Value, instructions []step) discord.Snowflake {
	// Use a type here to detect recursion:
	// var originalT = v.Type()
	var laststep reflect.Value

	log.Println(v.Type(), instructions)

	for i, step := range instructions {
		if !v.IsValid() {
			return 0
		}
		if i > 0 && step.ptr {
			v = v.Elem()
		}
		if !v.IsValid() {
			// is this the bottom of the instructions?
			if i == len(instructions)-1 && step.rec != nil {
				for _, ins := range step.rec {
					var value = laststep.Field(ins.field)
					if ins.ptr {
						value = value.Elem()
					}
					if id := applyInstructions(value, instructions); id.Valid() {
						return id
					}
				}
			}
			return 0
		}
		laststep = v
		v = laststep.Field(step.field)
	}
	return discord.Snowflake(v.Int())
}

func (r *reflector) instructions() []step {
	if len(r.steps) == 0 {
		return nil
	}
	var instructions = make([]step, len(r.steps))
	for i := 0; i < len(instructions); i++ {
		instructions[i] = r.steps[len(r.steps)-i-1]
	}
	// instructions := r.steps
	return instructions
}

func (r *reflector) step(s step) {
	r.steps = append(r.steps, s)
}

func (r *reflector) _id(v reflect.Value, t reflect.Type) (chID discord.Snowflake) {
	numFields := t.NumField()

	var ptr bool
	var ins = step{field: -1}

	for i := 0; i < numFields; i++ {
		field := t.Field(i)
		fType := field.Type
		value := v.Field(i)
		ptr = false

		if fType.Kind() == reflect.Ptr {
			fType = fType.Elem()
			value = value.Elem()
			ptr = true
		}

		// does laststep have the same field type?
		if fType == t {
			ins.rec = append(ins.rec, step{field: i, ptr: ptr})
		}

		if !value.IsValid() {
			continue
		}

		// If we've already found the field:
		if ins.field > 0 {
			continue
		}

		switch fType.Kind() {
		case reflect.Struct:
			if chID = r._id(value, fType); chID.Valid() {
				ins.field = i
				ins.ptr = ptr
			}
		case reflect.Int64:
			switch {
			case false,
				// Contains works with "LastMessageID" and such.
				strings.Contains(field.Name, r.thingID),
				// Special case where the struct name has Channel in it.
				field.Name == "ID" && strings.Contains(t.Name(), r.thing):

				ins.field = i
				ins.ptr = ptr

				chID = discord.Snowflake(value.Int())
			}
		}
	}

	// If we've found the field:
	r.step(ins)

	return
}
*/
