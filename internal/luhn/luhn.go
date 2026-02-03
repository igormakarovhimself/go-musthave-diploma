package luhn

import (
	"strconv"
	"strings"
)

func Validate(number string) bool {
	number = strings.ReplaceAll(number, " ", "")

	if number == "" {
		return false
	}

	sum := 0
	parity := len(number) % 2

	for i, char := range number {
		digit, err := strconv.Atoi(string(char))
		if err != nil {
			return false
		}

		if i%2 == parity {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}

		sum += digit
	}

	return sum%10 == 0
}
