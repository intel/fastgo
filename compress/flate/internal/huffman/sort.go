// Copyright (c) 2023, Intel Corporation.
// SPDX-License-Identifier: BSD-3-Clause

package huffman

import "unsafe"

func quickSortReverse(arr []uint32, left int, right int) {
	for left < right {
		if right-left < 16 {
			insertReverseSort(arr[left : right+1])
			break
		} else {
			pivot := arr[left]
			i := left + 1
			j := right

			for i <= j {
				if arr[i] < pivot && arr[j] > pivot {
					arr[i], arr[j] = arr[j], arr[i]
				}
				if arr[i] >= pivot {
					i++
				}
				if arr[j] <= pivot {
					j--
				}
			}

			arr[left], arr[j] = arr[j], arr[left]

			if j-left < right-j {
				quickSortReverse(arr, left, j-1)
				left = j + 1
			} else {
				quickSortReverse(arr, j+1, right)
				right = j - 1
			}
		}
	}
}

func insertReverseSort(arr []uint32) {
	for i := 1; i < len(arr); i++ {
		for j := i; j > 0 && arr[j-1] < arr[j]; j-- {
			arr[j-1], arr[j] = arr[j], arr[j-1]
		}
	}
}

func sortDecLitCounts(arr decLitCounts) {
	// force type conversion
	ilc := *(*[]uint32)(unsafe.Pointer(&arr))
	quickSortReverse(ilc, 0, len(arr)-1)
}
