// Copyright 2021 FerretDB Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package postgresql

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/FerretDB/FerretDB/internal/backends/postgresql/metadata"
	"github.com/FerretDB/FerretDB/internal/types"
	"github.com/FerretDB/FerretDB/internal/util/must"
)

func TestPrepareWhereClause(t *testing.T) {
	t.Parallel()
	objectID := types.ObjectID{0x62, 0x56, 0xc5, 0xba, 0x0b, 0xad, 0xc0, 0xff, 0xee, 0xff, 0xff, 0xff}

	// WHERE clauses occurring frequently in tests
	whereContain := " WHERE _jsonb->$1 @> $2"
	whereGt := " WHERE _jsonb->$1 > $2"
	whereNotEq := ` WHERE NOT ( _jsonb ? $1 AND _jsonb->$1 @> $2 AND _jsonb->'$s'->'p'->$1->'t' = `

	for name, tc := range map[string]struct {
		filter   *types.Document
		expected string
		skip     string
		args     []any // if empty, check is disabled
	}{
		"IDObjectID": {
			filter:   must.NotFail(types.NewDocument("_id", objectID)),
			expected: whereContain,
		},
		"IDString": {
			filter:   must.NotFail(types.NewDocument("_id", "foo")),
			expected: whereContain,
		},
		"IDBool": {
			filter:   must.NotFail(types.NewDocument("_id", "foo")),
			expected: whereContain,
		},
		"IDDotNotation": {
			filter: must.NotFail(types.NewDocument("_id.doc", "foo")),
		},

		"DotNotation": {
			filter: must.NotFail(types.NewDocument("v.doc", "foo")),
		},
		"DotNotationArrayIndex": {
			filter: must.NotFail(types.NewDocument("v.arr.0", "foo")),
		},

		"ImplicitString": {
			filter:   must.NotFail(types.NewDocument("v", "foo")),
			expected: whereContain,
		},
		"ImplicitEmptyString": {
			filter:   must.NotFail(types.NewDocument("v", "")),
			expected: whereContain,
		},
		"ImplicitInt32": {
			filter:   must.NotFail(types.NewDocument("v", int32(42))),
			expected: whereContain,
		},
		"ImplicitInt64": {
			filter:   must.NotFail(types.NewDocument("v", int64(42))),
			expected: whereContain,
		},
		"ImplicitFloat64": {
			filter:   must.NotFail(types.NewDocument("v", float64(42.13))),
			expected: whereContain,
		},
		"ImplicitMaxFloat64": {
			filter:   must.NotFail(types.NewDocument("v", math.MaxFloat64)),
			expected: whereGt,
		},
		"ImplicitBool": {
			filter:   must.NotFail(types.NewDocument("v", true)),
			expected: whereContain,
		},
		"ImplicitDatetime": {
			filter: must.NotFail(types.NewDocument(
				"v", time.Date(2021, 11, 1, 10, 18, 42, 123000000, time.UTC),
			)),
			expected: whereContain,
		},
		"ImplicitObjectID": {
			filter:   must.NotFail(types.NewDocument("v", objectID)),
			expected: whereContain,
		},

		"EqString": {
			filter: must.NotFail(types.NewDocument(
				"v", must.NotFail(types.NewDocument("$eq", "foo")),
			)),
			args:     []any{`v`, `"foo"`},
			expected: whereContain,
		},
		"EqEmptyString": {
			filter: must.NotFail(types.NewDocument(
				"v", must.NotFail(types.NewDocument("$eq", "")),
			)),
			expected: whereContain,
		},
		"EqInt32": {
			filter: must.NotFail(types.NewDocument(
				"v", must.NotFail(types.NewDocument("$eq", int32(42))),
			)),
			expected: whereContain,
		},
		"EqInt64": {
			filter: must.NotFail(types.NewDocument(
				"v", must.NotFail(types.NewDocument("$eq", int64(42))),
			)),
			expected: whereContain,
		},
		"EqFloat64": {
			filter: must.NotFail(types.NewDocument(
				"v", must.NotFail(types.NewDocument("$eq", float64(42.13))),
			)),
			expected: whereContain,
		},
		"EqMaxFloat64": {
			filter: must.NotFail(types.NewDocument(
				"v", must.NotFail(types.NewDocument("$eq", math.MaxFloat64)),
			)),
			args:     []any{`v`, types.MaxSafeDouble},
			expected: whereGt,
		},
		"EqDoubleBigInt64": {
			filter: must.NotFail(types.NewDocument(
				"v", must.NotFail(types.NewDocument("$eq", float64(2<<61))),
			)),
			args:     []any{`v`, types.MaxSafeDouble},
			expected: whereGt,
		},
		"EqBool": {
			filter: must.NotFail(types.NewDocument(
				"v", must.NotFail(types.NewDocument("$eq", true)),
			)),
			expected: whereContain,
		},
		"EqDatetime": {
			filter: must.NotFail(types.NewDocument(
				"v", must.NotFail(types.NewDocument(
					"$eq", time.Date(2021, 11, 1, 10, 18, 42, 123000000, time.UTC),
				)),
			)),
			expected: whereContain,
		},
		"EqObjectID": {
			filter: must.NotFail(types.NewDocument(
				"v", must.NotFail(types.NewDocument("$eq", objectID)),
			)),
			expected: whereContain,
		},

		"NeString": {
			filter: must.NotFail(types.NewDocument(
				"v", must.NotFail(types.NewDocument("$ne", "foo")),
			)),
			expected: whereNotEq + `'"string"' )`,
		},
		"NeEmptyString": {
			filter: must.NotFail(types.NewDocument(
				"v", must.NotFail(types.NewDocument("$ne", "")),
			)),
			expected: whereNotEq + `'"string"' )`,
		},
		"NeInt32": {
			filter: must.NotFail(types.NewDocument(
				"v", must.NotFail(types.NewDocument("$ne", int32(42))),
			)),
			expected: whereNotEq + `'"int"' )`,
		},
		"NeInt64": {
			filter: must.NotFail(types.NewDocument(
				"v", must.NotFail(types.NewDocument("$ne", int64(42))),
			)),
			expected: whereNotEq + `'"long"' )`,
		},
		"NeFloat64": {
			filter: must.NotFail(types.NewDocument(
				"v", must.NotFail(types.NewDocument("$ne", float64(42.13))),
			)),
			expected: whereNotEq + `'"double"' )`,
		},
		"NeMaxFloat64": {
			filter: must.NotFail(types.NewDocument(
				"v", must.NotFail(types.NewDocument("$ne", math.MaxFloat64)),
			)),
			args:     []any{`v`, math.MaxFloat64},
			expected: whereNotEq + `'"double"' )`,
		},
		"NeBool": {
			filter: must.NotFail(types.NewDocument(
				"v", must.NotFail(types.NewDocument("$ne", true)),
			)),
			expected: whereNotEq + `'"bool"' )`,
		},
		"NeDatetime": {
			filter: must.NotFail(types.NewDocument(
				"v", must.NotFail(types.NewDocument(
					"$ne", time.Date(2021, 11, 1, 10, 18, 42, 123000000, time.UTC),
				)),
			)),
			expected: whereNotEq + `'"date"' )`,
		},
		"NeObjectID": {
			filter: must.NotFail(types.NewDocument(
				"v", must.NotFail(types.NewDocument("$ne", objectID)),
			)),
			expected: whereNotEq + `'"objectId"' )`,
		},

		"Comment": {
			filter: must.NotFail(types.NewDocument("$comment", "I'm comment")),
		},
	} {
		name, tc := name, tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if tc.skip != "" {
				t.Skip(tc.skip)
			}

			actual, args, err := prepareWhereClause(new(metadata.Placeholder), tc.filter)
			require.NoError(t, err)

			assert.Equal(t, tc.expected, actual)

			if len(tc.args) == 0 {
				return
			}

			assert.Equal(t, tc.args, args)
		})
	}
}
