package luhn

import "testing"

func TestValidate(t *testing.T) {
	tests := []struct {
		name   string
		number string
		want   bool
	}{
		{
			name:   "valid number from spec",
			number: "12345678903",
			want:   true,
		},
		{
			name:   "valid number",
			number: "79927398713",
			want:   true,
		},
		{
			name:   "valid number with spaces",
			number: "7992 7398 713",
			want:   true,
		},
		{
			name:   "invalid checksum",
			number: "12345678904",
			want:   false,
		},
		{
			name:   "empty string",
			number: "",
			want:   false,
		},
		{
			name:   "contains letters",
			number: "1234567890a",
			want:   false,
		},
		{
			name:   "short valid number",
			number: "0",
			want:   true,
		},
		{
			name:   "another valid",
			number: "4532015112830366",
			want:   true,
		},
		{
			name:   "another invalid",
			number: "4532015112830367",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Validate(tt.number); got != tt.want {
				t.Errorf("Validate(%q) = %v, want %v", tt.number, got, tt.want)
			}
		})
	}
}
