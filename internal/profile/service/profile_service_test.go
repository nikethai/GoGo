package service

import (
	"testing"

	"main/internal/profile/model"
)

func TestProfileService_ValidateProfileUpdate(t *testing.T) {
	tests := []struct {
		name    string
		req     *model.ProfileUpdateRequest
		wantErr bool
	}{
		{
			name: "valid request",
			req: &model.ProfileUpdateRequest{
				Fullname: "John Doe",
				Email:    "john@example.com",
				Phone:    "1234567890",
				Address:  "123 Main St",
				DOB:      "1990-01-01",
			},
			wantErr: false,
		},
		{
			name: "empty fullname",
			req: &model.ProfileUpdateRequest{
				Fullname: "",
				Email:    "john@example.com",
				Phone:    "1234567890",
				Address:  "123 Main St",
				DOB:      "1990-01-01",
			},
			wantErr: true,
		},
		{
			name: "invalid email",
			req: &model.ProfileUpdateRequest{
				Fullname: "John Doe",
				Email:    "invalid-email",
				Phone:    "1234567890",
				Address:  "123 Main St",
				DOB:      "1990-01-01",
			},
			wantErr: true,
		},
		{
			name: "short phone",
			req: &model.ProfileUpdateRequest{
				Fullname: "John Doe",
				Email:    "john@example.com",
				Phone:    "123",
				Address:  "123 Main St",
				DOB:      "1990-01-01",
			},
			wantErr: true,
		},
		{
			name: "invalid date format",
			req: &model.ProfileUpdateRequest{
				Fullname: "John Doe",
				Email:    "john@example.com",
				Phone:    "1234567890",
				Address:  "123 Main St",
				DOB:      "invalid-date",
			},
			wantErr: true,
		},
	}

	// Create a service instance with nil dependencies for validation testing
	service := &ProfileService{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.ValidateProfileUpdate(tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateProfileUpdate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestProfileService_ValidateEmail(t *testing.T) {
	tests := []struct {
		name  string
		email string
		want  bool
	}{
		{"valid email", "test@example.com", true},
		{"valid email with subdomain", "user@mail.example.com", true},
		{"invalid email - no @", "testexample.com", false},
		{"invalid email - no domain", "test@", false},
		{"invalid email - no local part", "@example.com", false},
		{"empty email", "", false},
	}

	service := &ProfileService{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := service.isValidEmail(tt.email); got != tt.want {
				t.Errorf("isValidEmail() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProfileService_ValidatePhone(t *testing.T) {
	tests := []struct {
		name  string
		phone string
		want  bool
	}{
		{"valid phone 10 digits", "1234567890", true},
		{"valid phone 11 digits", "12345678901", true},
		{"valid phone 15 digits", "123456789012345", true},
		{"invalid phone - too short", "123456789", false},
		{"invalid phone - too long", "1234567890123456", false},
		{"invalid phone - contains letters", "123abc7890", false},
		{"empty phone", "", false},
	}

	service := &ProfileService{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := service.isValidPhone(tt.phone); got != tt.want {
				t.Errorf("isValidPhone() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProfileService_ValidateDOB(t *testing.T) {
	tests := []struct {
		name string
		dob  string
		want bool
	}{
		{"valid date", "1990-01-01", true},
		{"valid date leap year", "2000-02-29", true},
		{"invalid date format", "01-01-1990", false},
		{"invalid date format", "1990/01/01", false},
		{"invalid date", "1990-13-01", false},
		{"invalid date", "1990-02-30", false},
		{"empty date", "", false},
	}

	service := &ProfileService{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := service.isValidDOB(tt.dob); got != tt.want {
				t.Errorf("isValidDOB() = %v, want %v", got, tt.want)
			}
		})
	}
}