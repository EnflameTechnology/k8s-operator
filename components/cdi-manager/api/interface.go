// Copyright (c) 2024 Enflame. All Rights Reserved.

package api

import "gcu-operator/components/common/args"

type CDIHandler interface {
	SetCDIEnabled(args *args.PluginArgs) error
}
