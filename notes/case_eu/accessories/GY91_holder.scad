
// Show
gy91();


// Variables
$fn = 96;
PIN_RADIUS = 0.6; // Radius for Pi Pins /0.56 for a thight fit
WIDTH = 16.4;
LENGTH = 23;
THICKNESS = 1;
HOLDER_HEIGHT = 7;

// Main module

module gy91(){
    // Ground plate
    translate([0,0,0.5])
        cube([WIDTH, LENGTH, THICKNESS], center=true);

    // Sides
    translate([-7.7, 0, 1.9])
        cube([1, LENGTH, 2], center=true);
    translate([-7.55, 0, 3.4])
        cube([1.3, LENGTH, 1], center=true);
        
    translate([7.7, 0, 1.9])
        cube([1, LENGTH, 2], center=true);
    translate([7.2, 0, 3.4])
        cube([2, LENGTH, 1], center=true);
        
    // Pin holder
    difference() {
        translate([0, -12.8, 3])
            color([0, 1, 0])   
                cube([7.5, 2.5, 6], center=true);

        // Pin holes
        translate([0, -12.8, -0.51])
            cylinder(r=PIN_RADIUS, h=HOLDER_HEIGHT);
        translate([2.5, -12.8, -0.51])
            cylinder(r=PIN_RADIUS, h=HOLDER_HEIGHT);
        translate([-2.5, -12.8, -0.51])
            cylinder(r=PIN_RADIUS, h=HOLDER_HEIGHT);
    }
}
