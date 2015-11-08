/*
 * Minio Client (C) 2015 Minio, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"

	"github.com/minio/mc/pkg/console"
	. "gopkg.in/check.v1"
)

func (s *TestSuite) TestLSContext(c *C) {
	/// filesystem
	root, e := ioutil.TempDir(os.TempDir(), "cmd-")
	c.Assert(e, IsNil)
	defer os.RemoveAll(root)

	for i := 0; i < 10; i++ {
		objectPath := filepath.Join(root, "object"+strconv.Itoa(i))
		data := "hello"
		dataLen := len(data)
		err := putTarget(objectPath, int64(dataLen), bytes.NewReader([]byte(data)))
		c.Assert(err, IsNil)
	}

	for i := 0; i < 10; i++ {
		objectPath := server.URL + "/bucket/object" + strconv.Itoa(i)
		data := "hello"
		dataLen := len(data)
		err := putTarget(objectPath, int64(dataLen), bytes.NewReader([]byte(data)))
		c.Assert(err, IsNil)
	}

	e = app.Run([]string{os.Args[0], "ls", root})
	c.Assert(e, IsNil)
	c.Assert(console.IsError, Equals, false)

	// reset back
	console.IsExited = false

	e = app.Run([]string{os.Args[0], "ls", root + "..."})
	c.Assert(e, IsNil)
	c.Assert(console.IsError, Equals, false)

	// reset back
	console.IsExited = false

	e = app.Run([]string{os.Args[0], "ls", server.URL + "/bucket"})
	c.Assert(e, IsNil)
	c.Assert(console.IsError, Equals, false)

	// reset back
	console.IsExited = false

	e = app.Run([]string{os.Args[0], "ls", server.URL + "/bucket..."})
	c.Assert(e, IsNil)
	c.Assert(console.IsError, Equals, false)

	// reset back
	console.IsExited = false

	e = app.Run([]string{os.Args[0], "ls", server.URL + "/invalid"})
	c.Assert(e, IsNil)
	c.Assert(console.IsExited, Equals, true)

	// reset back
	console.IsExited = false
}
