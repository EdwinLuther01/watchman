// Copyright 2022 The Moov Authors
// Use of this source code is governed by an Apache License
// license that can be found in the LICENSE file.

package main

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"github.com/moov-io/base/log"
	"github.com/moov-io/watchman/pkg/ofac"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
)

func BenchmarkSearchHandler(b *testing.B) {
	searcher := createTestSearcher(b) // Uses live data
	b.ResetTimer()

	router := mux.NewRouter()
	addSearchRoutes(log.NewNopLogger(), router, searcher)

	g := &errgroup.Group{}
	g.SetLimit(10)

	for i := 0; i < b.N; i++ {
		g.Go(func() error {
			name := fake.Person().Name()

			v := make(url.Values, 0)
			v.Set("name", name)
			v.Set("limit", "10")
			v.Set("minMatch", "0.70")

			w := httptest.NewRecorder()
			req := httptest.NewRequest("GET", fmt.Sprintf("/search?%s", v.Encode()), nil)
			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				return fmt.Errorf("unexpected status: %v", w.Code)
			}
			return nil
		})
	}
	require.NoError(b, g.Wait())
}

func BenchmarkJaroWinkler(b *testing.B) {
	results, err := ofac.Read(filepath.Join("..", "..", "test", "testdata", "sdn.csv"))
	require.NoError(b, err)
	require.Len(b, results.SDNs, 7379)

	randomIndex := func(length int) int {
		n, err := rand.Int(rand.Reader, big.NewInt(1e9))
		if err != nil {
			panic(err)
		}
		return int(n.Int64()) % length
	}

	b.Run("bestPairsJaroWinkler", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			nameTokens := strings.Fields(fake.Person().Name())
			idx := randomIndex(len(results.SDNs))

			score := bestPairsJaroWinkler(nameTokens, results.SDNs[idx].SDNName)
			require.Greater(b, score, -0.01)
		}
	})
}

// goos: darwin
// goarch: amd64
// pkg: github.com/moov-io/watchman/cmd/server
// cpu: Intel(R) Core(TM) i9-9880H CPU @ 2.30GHz
// BenchmarkSearchHandler-16    	    2728	 131 213 518 ns/op	34812129 B/op	 1486792 allocs/op
// PASS
// ok  	github.com/moov-io/watchman/cmd/server	413.248s

// goos: darwin
// goarch: amd64
// pkg: github.com/moov-io/watchman/cmd/server
// cpu: Intel(R) Core(TM) i9-9880H CPU @ 2.30GHz
// BenchmarkSearchHandler-16    	    2079	 174 594 246 ns/op	49797019 B/op	 1638732 allocs/op
// PASS
// ok  	github.com/moov-io/watchman/cmd/server	419.284s
