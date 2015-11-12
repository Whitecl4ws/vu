// Copyright © 2013-2015 Galvanized Logic Inc.
// Use is governed by a BSD-style license found in the LICENSE file.
//
// Huge thanks to bullet physics for showing what a physics engine is all about
// in cool-hard-code reality rather than theory. Methods and files that were
// derived from bullet physics are commented to indicate their origin.
// Bullet physics, for the most part, has the following license:
//
//   Bullet Continuous Collision Detection and Physics Library
//   Copyright (c) 2003-2006 Erwin Coumans  http://continuousphysics.com/Bullet/
//
//   This software is provided 'as-is', without any express or implied warranty.
//   In no event will the authors be held liable for any damages arising from the use of this software.
//   Permission is granted to anyone to use this software for any purpose,
//   including commercial applications, and to alter it and redistribute it freely,
//   subject to the following restrictions:
//
//   1. The origin of this software must not be misrepresented; you must not claim that you wrote the original software.
//      If you use this software in a product, an acknowledgment in the product documentation would be appreciated but is not required.
//   2. Altered source versions must be plainly marked as such, and must not be misrepresented as being the original software.
//   3. This notice may not be removed or altered from any source distribution.

// Physics is a real-time simulation of real-world physics. Physics applies
// simulated forces to virtual 3D objects known as bodies. Physics updates
// bodies locations and directions based on forces and collisions with
// other bodies.
//
// Bodies are created using NewBody(shape). For example:
//    box    := NewBody(NewBox(hx, hy, hz))
//    sphere := NewBody(NewSphere(radius))
//
// Creating and storing bodies is the responsibility of the calling application.
// Bodies are moved with frequent and regular calls to Physics.Step().
// Regulating the calls to Step() is also the responsibility of the calling
// application. Once Step() has completed, the bodies updated location and
// direction are available in Body.World().
//
// Package physics is provided as part of the vu (virtual universe) 3D engine.
package physics

// See the open source physics engines:
//     www.bulletphysics.com
//     www.ode.org
// There is a 2D engine physics engine architecture overview at
//     http://gamedev.tutsplus.com/series/custom-game-physics-engine
// For regulating physics timesteps in the application see:
//     http://gafferongames.com/game-physics/fix-your-timestep
// Other physics references:
//     http://www.geometrictools.com/Source/Physics.html

// Physics simulates forces acting on moving bodies. Expected usage
// is to simulate real-life conditions like air resistance and gravity,
// or the lack thereof.
type Physics interface {
	SetGravity(gravity float64) // Default is 10m/s.
	SetMargin(margin float64)   // Default is 0.04.

	// Step the physics simulation one tiny bit forward. This is expected
	// to be called regularly from the main engine loop. At the end of
	// a simulation step, all the bodies positions will be updated based on
	// forces acting upon them and/or collision results. Unmoving/unmoved
	// bodies, or bodies with zero mass are not updated.
	Step(bodies []Body, timestep float64)

	// Collide checks for collision between bodies a, b independent of
	// the current physics simulation. Bodies positions and velocities
	// are not updated. Provided for occasional or one-off checks.
	Collide(a, b Body) bool
}

// Physics interface
// ===========================================================================
// physics: default Physics implementation.

// physics is the default implementation of the Physics interface.
// It coordinates the physics pipeline by calling broadphase,
// narrowphase, and solver.
type physics struct {
	gravity    float64                 // Force in m/s. Default is 10m/s.
	col        *collider               // Checks for collisions, updates collision contacts.
	sol        *solver                 // Resolves collisions, updates bodies locations.
	overlapped map[uint64]*contactPair // Overlapping pairs. Updated during broadphase.

	// scratch variables keep memory so that temp variables
	// don't have to be continually allocated and garbage collected
	abA, abB *Abox             // Scratch broadphase axis aligned bounding boxes.
	mf0      []*pointOfContact // Scratch narrowphase manifold.
}

// NewPhysics creates and returns a mover instance. Generally expected
// to be called once per application that needs a physics simulation.
func NewPhysics() Physics { return newPhysics() }
func newPhysics() *physics {
	px := &physics{}
	px.gravity = -10
	px.col = newCollider()
	px.sol = newSolver()
	px.overlapped = map[uint64]*contactPair{}
	px.mf0 = newManifold()
	px.abA = &Abox{}
	px.abB = &Abox{}
	return px
}

// margin is a gap for smoothing collision detections.
var margin float64 = 0.04

// maxFriction is used to limit the amount of friction that
// can be applied to the combined friction of colliding bodies.
var maxFriction float64 = 10.0

// Physics interface implementation.
// Step the physics simulation forward by delta time (timestep).
// Note that the body.iitw is initialized once the first pass completes.
func (px *physics) Step(bodies []Body, timestep float64) {

	// apply forces (e.g. gravity) to bodies and predict body locations
	px.predictBodyLocations(bodies, timestep)

	// update overlapped pairs
	px.broadphase(bodies, px.overlapped)
	if len(px.overlapped) > 0 {

		// collide overlapped pairs
		if colliding := px.narrowphase(px.overlapped); len(colliding) > 0 {
			px.sol.info.timestep = timestep

			// resolve all colliding pairs
			px.sol.solve(colliding, px.overlapped)
		}
	}

	// adjust body locations based on velocities
	px.updateBodyLocations(bodies, timestep)
	px.clearForces(bodies)
}

// Physics interface implementation.
func (px *physics) SetGravity(gravity float64)        { px.gravity = gravity }
func (px *physics) SetMargin(collisionMargin float64) { margin = collisionMargin }

// predictBodyLocations applies motion to moving/awake bodies as if there
// was nothing else around.
//
// Based on bullet btSimpleDynamicsWorld::predictUnconstraintMotion
func (px *physics) predictBodyLocations(bodies []Body, dt float64) {
	var b *body
	for _, bb := range bodies {
		b = bb.(*body)
		b.guess.Set(b.world)
		if b.movable {

			// Fg = m*a. Apply gravity as if mass was 1.
			// FUTURE: use bodies mass when applying gravity.
			b.applyGravity(px.gravity)     // updates forces.
			b.integrateVelocities(dt)      // applies forces to velocities.
			b.applyDamping(dt)             // damps velocities.
			b.updatePredictedTransform(dt) // applies velocities to prediction transform.
		}
	}
}

// broadphase checks for overlaps using the axis aligned bounding box
// for each body.
//
// FUTURE: create a broadphase bounding volume hierarchy to help with dealing
//         with a much larger number of bodies. Especially non-colliding bodies.
func (px *physics) broadphase(bodies []Body, pairs map[uint64]*contactPair) {
	for _, pair := range pairs {
		pair.valid = false // validate checks for deleted bodies.
	}
	var bodyA, bodyB *body
	var uniques []Body
	var pairId uint64
	for cnt1, B1 := range bodies {
		bodyA = B1.(*body)
		uniques = bodies[cnt1+1:]
		for _, B2 := range uniques {
			bodyB = B2.(*body)

			// FUTURE: Add masking feature that allows bodies to only collide
			//         with other bodies that have matching mask types.

			// check as long as one of the bodies can move.
			if bodyA.movable || bodyB.movable {
				pairId = bodyA.pairId(bodyB)
				pair, existing := pairs[pairId]
				if existing {
					pair.valid = true
					abA := bodyA.predictedAabb(px.abA, margin)
					abB := bodyB.predictedAabb(px.abB, margin)
					overlaps := abA.Overlaps(abB)
					if !overlaps {
						// Remove existing
						delete(pairs, pairId)
					}
					// Otherwise hold existing
				} else {
					abA := bodyA.worldAabb(px.abA)
					abB := bodyB.worldAabb(px.abB)
					overlaps := abA.Overlaps(abB)
					if overlaps {
						// Add new
						pair = newContactPair(bodyA, bodyB)
						pair.valid = true
						pairs[pairId] = pair
					}
					// Otherwise ignore non-overlapping pair
				}
			}
		}
	}

	// remove contact pairs referencing deleted bodies.
	for pairId, pair := range pairs {
		if !pair.valid {
			delete(pairs, pairId)
		}
	}
}

// narrowphase checks for actual collision. If bodies are colliding,
// then the persistent collision information for the bodies is updated.
// This includes the contact, normal, and depth information.
// Return all colliding bodies.
func (px *physics) narrowphase(pairs map[uint64]*contactPair) (colliding map[uint32]*body) {
	colliding = map[uint32]*body{}
	scrManifold := px.mf0 // scatch mf0
	for _, cpair := range pairs {
		bodyA, bodyB := cpair.bodyA, cpair.bodyB
		algorithm := px.col.algorithms[bodyA.shape.Type()][bodyB.shape.Type()]
		bA, bB, manifold := algorithm(bodyA, bodyB, scrManifold)
		cpair.bodyA, cpair.bodyB = bA.(*body), bB.(*body) // handle potential body swaps.

		// bodies are colliding if there are contact points in the manifold.
		// Update any contact points and prepare for the solver.
		if len(manifold) > 0 {
			colliding[bodyA.bid] = bodyA
			colliding[bodyB.bid] = bodyB
			cpair.refreshContacts(bodyA.world, bodyB.world)
			cpair.mergeContacts(manifold)
		}
	} // scratch mf0 free
	return colliding
}

// updateBodyLocations applies the updated linear and angular velocities to the
// the bodies current position.
func (px *physics) updateBodyLocations(bodies []Body, timestep float64) {
	var b *body
	for _, bb := range bodies {
		b = bb.(*body)
		if b.movable {
			b.updateWorldTransform(timestep)
			b.updateInertiaTensor()
		}
	}
}

// clearFoces removes any forces acting on bodies. This allows for the forces
// to be changed each simulation step.
func (px *physics) clearForces(bodies []Body) {
	var b *body
	for _, bb := range bodies {
		b = bb.(*body)
		b.clearForces()
	}
}

// Collide returns true if the two shapes, a, b are touching or overlapping.
func (px *physics) Collide(a, b Body) (hit bool) {
	aa, bb := a.(*body), b.(*body)
	algorithm := px.col.algorithms[aa.shape.Type()][bb.shape.Type()]
	_, _, manifold := algorithm(aa, bb, px.mf0)
	return len(manifold) > 0
}

// Cast checks if a ray r intersects the given Form f, giving back the
// nearest point of intersection if there is one. The point of contact
// x, y, z is valid when hit is true.
func Cast(ray, b Body) (hit bool, x, y, z float64) {
	if ray != nil && b != nil && b.Shape() != nil {
		if alg, ok := rayCastAlgorithms[b.Shape().Type()]; ok {
			return alg(ray, b)
		}
	}
	return false, 0, 0, 0
}
