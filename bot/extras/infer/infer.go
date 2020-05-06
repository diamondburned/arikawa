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
			if field.Name == thing+"ID" {
				// grab value real quick
				return discord.Snowflake(v.Field(i).Int())
			}

			// Special case where the struct name has Channel in it
			if field.Name == "ID" && strings.Contains(t.Name(), thing) {
				return discord.Snowflake(v.Field(i).Int())
			}
		}
	}

	return 0
}
