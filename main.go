/*
 * Copyright 2018-2020, VMware, Inc. All Rights Reserved.
 * Proprietary and Confidential.
 * Unauthorized use, copying or distribution of this source code via any medium is
 * strictly prohibited without the express written consent of VMware, Inc.
 */

package libpak

import (
	"github.com/buildpacks/libcnb"
	"github.com/paketo-buildpacks/libpak/internal"
)

// Main is called by the main function of a buildpack, encapsulating both detection and build in the same binary.
func Main(detector libcnb.Detector, builder libcnb.Builder, options ...libcnb.Option) {
	libcnb.Main(detectDelegate{delegate: detector}, buildDelegate{delegate: builder},
		append([]libcnb.Option{
			libcnb.WithEnvironmentWriter(internal.NewEnvironmentWriter()),
			libcnb.WithExitHandler(internal.NewExitHandler()),
			libcnb.WithTOMLWriter(internal.NewTOMLWriter()),
		}, options...)...,
	)
}
