package main

import "strings"

func splitSemicolon(s string) []string { return strings.Split(s, ";") }
func indexByte(s string, b byte) int   { return strings.IndexByte(s, b) }
func trimSpace(s string) string        { return strings.TrimSpace(s) }
