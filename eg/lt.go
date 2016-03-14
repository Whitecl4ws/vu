// Copyright © 2015-2016 Galvanized Logic Inc.
// Use is governed by a BSD-style license found in the LICENSE file.

package main

import (
	"log"

	"github.com/gazed/vu"
	"github.com/gazed/vu/math/lin"
)

// lt tests the engines handling of some of the engine lighting shaders.
// It also checks the conversion of light position and normal vectors
// needed for proper lighting.
//
// Note the use of the box.obj model that needs 24 verticies to get
// proper lighting on each face. Also note how many more verticies are
// necessary for the sphere.obj model.
func lt() {
	lt := &lttag{}
	if err := vu.New(lt, "Lighting", 400, 100, 800, 600); err != nil {
		log.Printf("lt: error starting engine %s", err)
	}
	defer catchErrors()
}

// Globally unique "tag" that encapsulates example specific data.
type lttag struct {
	cam3D vu.Camera // 3D main scene camera.
	sun   vu.Pov    // Light node in Pov hierarchy.
}

// Create is the engine callback for initial asset creation.
func (lt *lttag) Create(eng vu.Eng, s *vu.State) {
	top := eng.Root().NewPov()
	lt.cam3D = top.NewCam()
	lt.cam3D.SetLocation(0.5, 2, 0.5)
	lt.sun = top.NewPov().SetLocation(0, 2.5, -1.75).SetScale(0.05, 0.05, 0.05)
	lt.sun.NewLight().SetColor(0.4, 0.7, 0.9)

	// Model at the light position.
	lt.sun.NewModel("solid").LoadMesh("sphere").LoadMat("red")

	// Create solid spheres to test the lighting shaders.
	c4 := top.NewPov().SetLocation(-0.5, 2, -2).SetScale(0.25, 0.25, 0.25)
	c4.NewModel("diffuse").LoadMesh("sphere").LoadMat("gray")
	c5 := top.NewPov().SetLocation(0.5, 2, -2).SetScale(0.25, 0.25, 0.25)
	c5.NewModel("gouraud").LoadMesh("sphere").LoadMat("gray")
	c6 := top.NewPov().SetLocation(1.5, 2, -2).SetScale(0.25, 0.25, 0.25)
	c6.NewModel("phong").LoadMesh("sphere").LoadMat("gray")

	// place and angle a large flat box behind the spheres.
	wall := top.NewPov().SetLocation(0, 2, -10).SetScale(5, 5, 5)
	wall.Spin(45, 45, 0)
	wall.NewModel("diffuse").LoadMesh("box").LoadMat("gray")
	lt.resize(s.W, s.H)
}

// Update is the regular engine callback.
func (lt *lttag) Update(eng vu.Eng, in *vu.Input, s *vu.State) {
	run := 10.0 // move so many units worth in one second.
	if in.Resized {
		lt.resize(s.W, s.H)
	}
	// move the light.
	dt := in.Dt
	speed := run * dt * 0.5
	for press, _ := range in.Down {
		switch press {
		case vu.K_W:
			lt.sun.Move(0, 0, -speed, lin.QI) // forward
		case vu.K_S:
			lt.sun.Move(0, 0, speed, lin.QI) // back
		case vu.K_A:
			lt.sun.Move(-speed, 0, 0, lin.QI) // left
		case vu.K_D:
			lt.sun.Move(speed, 0, 0, lin.QI) // right
		case vu.K_Z:
			lt.sun.Move(0, speed, 0, lin.QI) // up
		case vu.K_X:
			lt.sun.Move(0, -speed, 0, lin.QI) // down
		}
	}
}
func (lt *lttag) resize(ww, wh int) {
	lt.cam3D.SetPerspective(60, float64(ww)/float64(wh), 0.1, 50)
}
