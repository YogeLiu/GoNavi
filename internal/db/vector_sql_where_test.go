package db

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

type vectorWhereRoundTripFunc func(*http.Request) (*http.Response, error)

func (fn vectorWhereRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func TestVectorSQLWhereConversion(t *testing.T) {
	expr, found, err := parseVectorSQLWhere(`SELECT * FROM products WHERE (category = 'book' OR price < 5) AND active != false LIMIT 10;`)
	if err != nil || !found {
		t.Fatalf("parseVectorSQLWhere() = (%#v, %v, %v)", expr, found, err)
	}

	chromaJSON, _ := json.Marshal(chromaWhereFromExpr(expr))
	for _, fragment := range []string{`"$and"`, `"$or"`, `"category":{"$eq":"book"}`, `"price":{"$lt":5}`, `"active":{"$ne":false}`} {
		if !strings.Contains(string(chromaJSON), fragment) {
			t.Errorf("Chroma filter %s missing %s", chromaJSON, fragment)
		}
	}

	qdrantJSON, _ := json.Marshal(qdrantFilterFromExpr(expr))
	for _, fragment := range []string{`"must"`, `"should"`, `"key":"category"`, `"match":{"value":"book"}`, `"range":{"lt":5}`, `"must_not"`} {
		if !strings.Contains(string(qdrantJSON), fragment) {
			t.Errorf("Qdrant filter %s missing %s", qdrantJSON, fragment)
		}
	}
}

func TestVectorSQLWhereRejectsUnsupportedSyntax(t *testing.T) {
	queries := []string{
		`SELECT * FROM products WHERE category LIKE 'book%'`,
		`SELECT * FROM products WHERE price BETWEEN 1 AND 5`,
		`SELECT * FROM products WHERE id IN (1, 2)`,
		`SELECT * FROM products WHERE category = NULL`,
		`SELECT * FROM products WHERE (active = true`,
		`SELECT * FROM products WHERE active = true ORDER BY price`,
	}
	for _, query := range queries {
		if _, found, err := parseVectorSQLWhere(query); !found || err == nil {
			t.Errorf("parseVectorSQLWhere(%q) found=%v err=%v, want explicit error", query, found, err)
		}
	}
}
