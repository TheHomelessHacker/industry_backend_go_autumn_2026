package main

func rotateRunes(s string, shift int) string {
	runes := []rune(s)	
	len_ := len(runes)

	if len_ == 0 {
		return ""
	}

	shift = shift % len_
	if shift < 0 {
		shift += len_
	}

	left := runes[:shift]
	right := runes[shift:]
	final := append(right,left...)
	return string(final)
}