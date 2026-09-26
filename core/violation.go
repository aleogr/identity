// SPDX-License-Identifier: Apache-2.0

package core

import postgres "github.com/aleogr/identity/adapters/postgres" // deliberate: shows depguard refusing it

var _ = postgres.X
