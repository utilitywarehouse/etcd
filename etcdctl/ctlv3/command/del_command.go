// Copyright 2015 The etcd Authors
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

package command

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"go.etcd.io/etcd/clientv3"
)

var (
	delPrefix      bool
	delPrevKV      bool
	delFromKey     bool
	delKeyContains string
	delExecute     bool
)

// NewDelCommand returns the cobra command for "del".
func NewDelCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "del [options] <key> [range_end]",
		Short: "Removes the specified key or range of keys [key, range_end)",
		Run:   delCommandFunc,
	}

	cmd.Flags().BoolVar(&delPrefix, "prefix", false, "delete keys with matching prefix")
	cmd.Flags().BoolVar(&delPrevKV, "prev-kv", false, "return deleted key-value pairs")
	cmd.Flags().BoolVar(&delFromKey, "from-key", false, "delete keys that are greater than or equal to the given key using byte compare")
	cmd.Flags().BoolVar(&delExecute, "execute", false, "only print what keys will be deleted - only for key-contains")
	cmd.Flags().StringVar(&delKeyContains, "key-contains", "", "delete keys that contain the matching string")
	return cmd
}

// delCommandFunc executes the "del" command.
func delCommandFunc(cmd *cobra.Command, args []string) {
	key, opts := getDelOp(args)
	ctx, cancel := commandCtx(cmd)
	defer cancel()
	client := mustClientFromCmd(cmd)
	if len(delKeyContains) > 0 {
		getResp, err := client.Get(ctx, "\x00", clientv3.WithRange(""), clientv3.WithFromKey())
		if err != nil {
			ExitWithError(ExitError, err)
		}
		for _, kv := range getResp.Kvs {
			sk := string(kv.Key)
			if strings.Contains(sk, delKeyContains) {
				fmt.Printf("Found Key %s. Binary is: % x\n", sk, kv.Key)
				if delExecute {
					fmt.Printf("deleting key...\n")
					resp, err := client.Delete(ctx, string(kv.Key))
					if err != nil {
						ExitWithError(ExitError, err)
					}
					display.Del(*resp)
				}
			}
		}
	} else { // normal flow
		resp, err := client.Delete(ctx, key, opts...)
		if err != nil {
			ExitWithError(ExitError, err)
		}
		display.Del(*resp)
	}
}

func getDelOp(args []string) (string, []clientv3.OpOption) {
	if len(args) == 0 || len(args) > 2 {
		ExitWithError(ExitBadArgs, fmt.Errorf("del command needs one argument as key and an optional argument as range_end"))
	}

	if delPrefix && delFromKey {
		ExitWithError(ExitBadArgs, fmt.Errorf("`--prefix` and `--from-key` cannot be set at the same time, choose one"))
	}

	opts := []clientv3.OpOption{}
	key := args[0]
	if len(args) > 1 {
		if delPrefix || delFromKey {
			ExitWithError(ExitBadArgs, fmt.Errorf("too many arguments, only accept one argument when `--prefix` or `--from-key` is set"))
		}
		opts = append(opts, clientv3.WithRange(args[1]))
	}

	if delPrefix {
		if len(key) == 0 {
			key = "\x00"
			opts = append(opts, clientv3.WithFromKey())
		} else {
			opts = append(opts, clientv3.WithPrefix())
		}
	}
	if delPrevKV {
		opts = append(opts, clientv3.WithPrevKV())
	}

	if delFromKey {
		if len(key) == 0 {
			key = "\x00"
		}
		opts = append(opts, clientv3.WithFromKey())
	}

	return key, opts
}
