/*
 * Copyright 2020 Simon Edwards <simon@simonzone.com>
 *
 * This source code is licensed under the MIT license which is detailed in the LICENSE.txt file.
 */
package internalpty

type InternalPty interface {
	Terminate()
	Write(data string)
	Resize(rows, cols int)
	PermitDataSize(size int)
	GetWorkingDirectory() string
}
