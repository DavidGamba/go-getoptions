// This file is part of go-getoptions.
//
// Copyright (C) 2015-2025  David Gamba Rios
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package getoptions

import (
	"errors"
	"fmt"
)

// ErrorHelpCalled - Indicates the help has been handled.
var ErrorHelpCalled = fmt.Errorf("help called")

// ErrorParsing - Indicates that there was an error with cli args parsing
var ErrorParsing = errors.New("")

// ErrorNotFound - Generic not found error
var ErrorNotFound = fmt.Errorf("not found")
