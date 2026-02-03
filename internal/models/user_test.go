package models

import (
	"testing"
)

func TestUser_BeforeSave_ValidGender(t *testing.T) {
	tests := []struct {
		name    string
		gender  string
		wantErr bool
	}{
		{
			name:    "Male gender",
			gender:  GenderMale,
			wantErr: false,
		},
		{
			name:    "Female gender",
			gender:  GenderFemale,
			wantErr: false,
		},
		{
			name:    "Invalid gender",
			gender:  "other",
			wantErr: true,
		},
		{
			name:    "Empty gender",
			gender:  "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := &User{
				TelegramID: 123456789,
				FullName:   "Test User",
				Gender:     tt.gender,
				Age:        25,
				City:       "Tehran",
				Status:     UserStatusOffline,
			}

			err := user.BeforeSave(nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("BeforeSave() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUser_BeforeSave_ValidAge(t *testing.T) {
	tests := []struct {
		name    string
		age     int
		wantErr bool
	}{
		{
			name:    "Minimum valid age",
			age:     13,
			wantErr: false,
		},
		{
			name:    "Normal age",
			age:     25,
			wantErr: false,
		},
		{
			name:    "Maximum valid age",
			age:     100,
			wantErr: false,
		},
		{
			name:    "Too young",
			age:     12,
			wantErr: true,
		},
		{
			name:    "Too old",
			age:     101,
			wantErr: true,
		},
		{
			name:    "Negative age",
			age:     -1,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := &User{
				TelegramID: 123456789,
				FullName:   "Test User",
				Gender:     GenderMale,
				Age:        tt.age,
				City:       "Tehran",
				Status:     UserStatusOffline,
			}

			err := user.BeforeSave(nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("BeforeSave() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUser_BeforeSave_ValidStatus(t *testing.T) {
	tests := []struct {
		name    string
		status  string
		wantErr bool
	}{
		{
			name:    "Offline status",
			status:  UserStatusOffline,
			wantErr: false,
		},
		{
			name:    "Online status",
			status:  UserStatusOnline,
			wantErr: false,
		},
		{
			name:    "Searching status",
			status:  UserStatusSearching,
			wantErr: false,
		},
		{
			name:    "In match status",
			status:  UserStatusInMatch,
			wantErr: false,
		},
		{
			name:    "Invalid status",
			status:  "invalid",
			wantErr: true,
		},
		{
			name:    "Empty status",
			status:  "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := &User{
				TelegramID: 123456789,
				FullName:   "Test User",
				Gender:     GenderMale,
				Age:        25,
				City:       "Tehran",
				Status:     tt.status,
			}

			err := user.BeforeSave(nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("BeforeSave() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUser_TableName(t *testing.T) {
	user := User{}
	tableName := user.TableName()

	if tableName != "users" {
		t.Errorf("TableName() = %q, want %q", tableName, "users")
	}
}

func TestUserConstants(t *testing.T) {
	// Test gender constants
	if GenderMale != "male" {
		t.Errorf("GenderMale = %q, want %q", GenderMale, "male")
	}
	if GenderFemale != "female" {
		t.Errorf("GenderFemale = %q, want %q", GenderFemale, "female")
	}

	// Test status constants
	if UserStatusOffline != "offline" {
		t.Errorf("UserStatusOffline = %q, want %q", UserStatusOffline, "offline")
	}
	if UserStatusOnline != "online" {
		t.Errorf("UserStatusOnline = %q, want %q", UserStatusOnline, "online")
	}
	if UserStatusSearching != "searching" {
		t.Errorf("UserStatusSearching = %q, want %q", UserStatusSearching, "searching")
	}
	if UserStatusInMatch != "in_match" {
		t.Errorf("UserStatusInMatch = %q, want %q", UserStatusInMatch, "in_match")
	}
}
