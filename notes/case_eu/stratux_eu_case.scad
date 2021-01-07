// HOW TO USE:
// Open this file in OpenScad to view or modify it. Hitting F5 will preview the file, F6 to render, then it can be exported.
// For viewing, you will want to uncomment one of these calls, depending on what you want to see

//case();
//lid();
view_side_by_side();
//view_assembled(gap=0.2);

// You can modify the parameters below to get your own custom, slightly modified case


POWER_CONNECTOR_HOLE_WIDTH = 12;
POWER_CONNECTOR_HOLE_HEIGHT = 8;

// Provide lead-through for battery pack / velcro strap?
STRAP_HOLDER = false;
STRAP_HOLDER_WIDTH = 20;
STRAP_HOLDER_HEIGHT = 3;
STRAP_HOLDER_OFFSET = 5;

// Set this to "3", "4" or "HYBRID". The only difference is a slighly moved
// power connector
RASPI_VERSION = "HYBRID";


// wall and base/top thickness
WALL_THICKNESS = 1.8;

OUTSIDE_CORNER_RADIUS = 1.5;

// Use "40", "30" or "NONE" to decide which fan you want to include
FAN_TYPE="40";

// Standard T-Beam GPS antenna is 16x6mm. If you decide to use a larger antenna,
// change the hole size here. This is WITHOUT tolerance.
GPS_ANTENNA_HOLE_SIZE = [16, 6]; 

// 0.3mm should work on most printers, but you might have small gaps between parts.
// If you have a good printer, go for 0.2 or even 0.15.
PRINTING_TOLERANCE_XY = 0.2;

// Most printers are more precise in the Z direction
PRITING_TOLERANCE_Z = 0.15;

// This defines the resolution of the resulting meshe's round elements
$fn=70;


// dimensions of the rPI part
_case_main_width = 64 + 2 * WALL_THICKNESS;
_case_main_length = 108 + WALL_THICKNESS;
_case_height = 25;

// dimensions of the T-Beam part
// Important: these are INCLUDING 0.3mm tolerance on each size and 0.1mm in thickness
_tbeam_pcb_width = 100.13 + 2 * PRINTING_TOLERANCE_XY;
_tbeam_pcb_length = 32.89 + 2 * PRINTING_TOLERANCE_XY;
_tbeam_pcb_thickness = 1.7 + 2 * PRITING_TOLERANCE_Z;
_tbeam_corner_radius = 1.3;


// PCB size + walls + tolerance.
// We only use 1*WALL_THICKNESS here, because the base should not be able to fit the T-Beam inside.
// Instead, the T-Beam should rest on the base.
// The lid then uses thinner walls (WALL_THICKNESS/2) around the T-Beam to cover it
_case_wing_width = _tbeam_pcb_width + 1 * WALL_THICKNESS;
_case_wing_length = _tbeam_pcb_length + 1 * WALL_THICKNESS;







// DO NOT MODIFY

// how much the wing stands out on either side
_wing_part_width = (_case_wing_width - _case_main_width) / 2;
_case_total_length = _case_main_length + _case_wing_length;




module roundedcube(size, center=false, rx=0, ry=0, rz=0) {
    
    translation = center ? [0,0,0] : [size[0]/2, size[1]/2, size[2]/2];
    
    if (rx == 0 && ry == 0 && rz == 0) {
        cube(size=size, center=center);
    } else {
        translate(translation) {
            intersection() {
                
                // Extrude upwards for the z radius
                if (rz > 0) {
                    linear_extrude(size[2], center=true)
                        offset(r=rz) square([size[0] - 2*rz, size[1] - 2*rz], center=true);
                }

                
                if (rx > 0) {
                    // Extrude for the x radius
                    rotate([90, 0, 90])
                        linear_extrude(size[0], center=true) 
                            offset(r=rx) square([size[1] - 2*rx, size[2] - 2*rx], center=true);
                }
                
                if (ry > 0) {
                    // Extrude for the y radius
                    rotate([90, 0, 0])
                        linear_extrude(size[1], center=true) 
                            offset(r=ry) square([size[0] - 2*ry, size[2] - 2*ry], center=true);
                }
            }
        }
    }
}

// Helper module to create the basic T-like shape that we use. Used to construct case and lid
module base_case_shape(top_thickness, side_thickness, height) {
    difference() {
        union() {
            // slightly longer for flat corners at the joint to the wings
            roundedcube([_case_main_width, _case_main_length+2, height], rz=OUTSIDE_CORNER_RADIUS);
            translate([-_wing_part_width, _case_main_length])
                roundedcube([_case_wing_width, _case_wing_length, height], rz=OUTSIDE_CORNER_RADIUS);
        }
        
        // make main part hollow hollow
        translate([side_thickness, side_thickness, top_thickness]) {
            roundedcube([_case_main_width-2*side_thickness, _case_main_length + 2, height], rz=OUTSIDE_CORNER_RADIUS);
        }
        
        // make T-Beam part hollow
        translate([-_wing_part_width + side_thickness, _case_main_length + side_thickness, top_thickness]) {
            roundedcube([_case_wing_width - 2 * side_thickness, _case_wing_length - 2*side_thickness, height], rz=OUTSIDE_CORNER_RADIUS);

        }
       
    }
    
}

module base_case() {
    base_case_shape(WALL_THICKNESS, WALL_THICKNESS, _case_height);
}

module pi_screw_mount() {
    union() {
        fillet_radius = 1.2;
        // Creates a fillet around the base for more stability
        rotate_extrude() {
            translate([2.99, 0, 0])
            difference() {
                square(fillet_radius);
                translate([fillet_radius, fillet_radius]) circle(fillet_radius);
            }
        }
        cylinder(3, 3, 3);
    }
}

module pi_screw_mounts() {
    // PI screw hole distances are 58x49mm
    translate([-49/2, -58/2]) pi_screw_mount();
    translate([-49/2, 58/2]) pi_screw_mount();
    translate([49/2, 58/2]) pi_screw_mount();
    translate([49/2, -58/2]) pi_screw_mount();
}

module pi_screw_hole() {
    translate([0, 0, -1]) cylinder(4.01, 1.2, 1.2);
}

module pi_screw_holes() {
    translate([-49/2, -58/2]) pi_screw_hole();
    translate([-49/2, 58/2]) pi_screw_hole();
    translate([49/2, 58/2]) pi_screw_hole();
    translate([49/2, -58/2]) pi_screw_hole();
}


module power_connector_hole() {
    radius = WALL_THICKNESS / 2;
    translate([-WALL_THICKNESS - 0.01, -POWER_CONNECTOR_HOLE_WIDTH / 2, -POWER_CONNECTOR_HOLE_HEIGHT / 2])
        roundedcube([WALL_THICKNESS + 0.02, POWER_CONNECTOR_HOLE_WIDTH, POWER_CONNECTOR_HOLE_HEIGHT],
                     rx=1);
}

module strap_holder() {
    radius = WALL_THICKNESS / 2;
    translate([-WALL_THICKNESS - 0.01, -STRAP_HOLDER_WIDTH / 2, -STRAP_HOLDER_HEIGHT / 2])
        roundedcube([WALL_THICKNESS + 0.02, STRAP_HOLDER_WIDTH, STRAP_HOLDER_HEIGHT],
                     rx=1);
}

module sd_card_hole() {
    translate([-6, -WALL_THICKNESS/2 - 0.01, -5.01])
        roundedcube([12.01, WALL_THICKNESS + 1.0, 10.01], ry=1);
}


module sma_connector_hole() {
    translate([0, -0.01])
        rotate([0, 90, 90])
            cylinder(h=WALL_THICKNESS + 0.02, r=3.35);
}

module corner_screw_holder(rotation_deg, holeradius, holexoffset, holeyoffset, cornersize=6) {
    rotate([0, 0, rotation_deg]) {
        difference() {
            // corner notch
            linear_extrude(height=10) 
                polygon([[0,0], [cornersize,0], [0,cornersize]]);
            
            // screw hole
            // We leave a single layer (0.3mm) on the bottom, so that
            // 3d printer can do some bridging easily
            translate([holexoffset, holeyoffset, 0.3])
                cylinder(10.00, r=holeradius);
        }
    }
}

module vent_holes() {
    // The cooling vent holes that are used for air-intage/exhaust on the back side
    // double as SMA connector holes. They are aligned to where the SMA connector is located on T-Beams.
    // (center for T-Beam 0.7, off-center for T-Beam 1.0 and non-fixed via cable on newer revisions
    for (i = [0 : 11.2 : 32]) {
        translate([i, 0, 0]) sma_connector_hole();
        translate([-i, 0, 0]) sma_connector_hole();
    }
}

// MAIN MODULE TO BUILD THE CASE
module case() {
    // See here for exact dimensions of the pi:
    // https://www.raspiworld.com/images/other/drawings/Raspberry-Pi-1-2-3-Model-B.pdf
    // We use a bit of tolerance for the case (e.g. 0.5mm from the left side)
    intersection() {
        difference() {
            union() {
                base_case();
                // Create PI screw pillars
                translate([_case_main_width/2, 58/2 + WALL_THICKNESS + 4, WALL_THICKNESS])
                    pi_screw_mounts();
                
                // screw holders for the lid. 
                translate([WALL_THICKNESS, WALL_THICKNESS, _case_height - 10])
                    corner_screw_holder(0, 0.8, 1.4, 1.4);
                translate([_case_main_width-WALL_THICKNESS, WALL_THICKNESS, _case_height - 10])
                    corner_screw_holder(90, 0.8, 1.4, 1.4);
                
                
                // Note that the T-Beam screws are _not_ symmetric.
                // On the side where the GPS chip is, distance from from side is 2.36mm (to hole-center).
                // On the other side, it is 2.67mm Thistens to the longer sides is 2.59 all around
                // Additionally, add 0.3mm tolerance
                offset_gps_side = 2.36 + PRINTING_TOLERANCE_XY - WALL_THICKNESS / 2;
                offset_pin_side = 2.67 + PRINTING_TOLERANCE_XY - WALL_THICKNESS / 2;
                offset_other = 2.59 + PRINTING_TOLERANCE_XY - WALL_THICKNESS / 2;
                
                translate([-_wing_part_width+WALL_THICKNESS, _case_main_length+WALL_THICKNESS, _case_height - 10])
                    corner_screw_holder(0, 0.8, offset_gps_side, offset_other, cornersize=7.5);
                
                translate([_case_main_width + _wing_part_width - WALL_THICKNESS, _case_main_length+WALL_THICKNESS, _case_height - 10])
                    corner_screw_holder(90, 0.8, offset_pin_side, offset_other, cornersize=7.5);
            }
            // clear holes for PI screws
            translate([_case_main_width/2, 58/2 + WALL_THICKNESS + 4, WALL_THICKNESS])
                pi_screw_holes();
            

            // clear hole for power connector
            // left side: older PI have the power connector 10.6mm in, pi4 has 12.4mm.
            // For hybrid we use the average 11.5
            // -> 11.5 + 0.5 gap from the side
            // height: wall + pillar height + pcb height + socket height/2
            connector_offset = RASPI_VERSION == "3" ? 10.6 :
                RASPI_VERSION == "4" ? 12.4 :
                11.5;

            translate([_case_main_width, WALL_THICKNESS + 0.5 + connector_offset, WALL_THICKNESS + 3 + 1.5 + 3.3/2])
                power_connector_hole();


            // clear holes for battery pack strap
            if (STRAP_HOLDER) {
                strap_offset = 60;

                translate([_case_main_width, WALL_THICKNESS + 0.5 + strap_offset, WALL_THICKNESS + _case_height - STRAP_HOLDER_OFFSET - 3/2])
                    strap_holder();

                translate([WALL_THICKNESS, WALL_THICKNESS + 0.5 + strap_offset, WALL_THICKNESS + _case_height - STRAP_HOLDER_OFFSET - 3/2])
                    strap_holder();
            }

            // clear hole for SD card slot
            translate([_case_main_width / 2, 0, 0])
                sd_card_hole();
            

            
            // The two SMA connectors
            translate([-8, _case_main_length, WALL_THICKNESS + 8])
                sma_connector_hole();
            translate([_case_main_width + 8, _case_main_length, WALL_THICKNESS + 8])
                sma_connector_hole();
            

            // Remove vent holes (double-SMA-connector-holes for T-Beam antenna)
            translate([_case_main_width / 2, _case_total_length - WALL_THICKNESS, _case_height - 6.8 + PRITING_TOLERANCE_Z])
                vent_holes();
        }
    }
}

module tbeam() {
    // Dimensions: see here. We leave 0.3mm gap on each side, so these numbers are a bit larger
    roundedcube([_tbeam_pcb_width, _tbeam_pcb_length, _tbeam_pcb_thickness], rz=_tbeam_corner_radius);
}

module lid_screw_hole() {
    translate([0, 0, -0.01]) cylinder(h=10, r=1.1);
}

module gps_antenna_hole() {
    square([GPS_ANTENNA_HOLE_SIZE[0] + 2 * PRINTING_TOLERANCE_XY, GPS_ANTENNA_HOLE_SIZE[1] + 2 * PRINTING_TOLERANCE_XY], center=true);
}

// 40mm fan: fan(holedist=32, outercircle=19, innercircle=10)
// 30mm fan: fan(holedist=24, outercircle=14, innercircle=9)
module fan(holedist=32, outercircle=19, innercircle=10) {
    holeoffset = holedist / 2;
    // Screw holes
    translate([-holeoffset, -holeoffset]) circle(1.3);
    translate([-holeoffset, holeoffset]) circle(1.3);
    translate([holeoffset, -holeoffset]) circle(1.3);
    translate([holeoffset, holeoffset]) circle(1.3);
    
    difference() {
        circle(outercircle);
        circle(innercircle);
        
        // Now remove a cross, so the guard in the middle has something it is attached to
        rotate(45) {
            square([2, 100], center = true);
            square([100, 2], center = true);
        }    
    }
}


// MAIN MODULE TO BUILD THE LID
module lid() {
    difference() {
        lid_height = WALL_THICKNESS + 1.5 + _tbeam_pcb_thickness;
        union() {
            
            // lid height will be thickness of the base + 1.5mm space for soldering joints on the T-Beam
            // + thickness of the PCB (incl. tolerances)
            
            base_case_shape(WALL_THICKNESS, WALL_THICKNESS + 1 + PRINTING_TOLERANCE_XY, lid_height);
            
            // lid guides so that the lid fits nicely on the base and gaps are not so visible
            // the +- 6 or 12 is to leave space for the screw holders in the main case
            tolerance = PRINTING_TOLERANCE_XY;
            
            // TODO: no idea why lid_height - 0.001 is needed. Otherwise the STL export will have the 
            // guides exported as seperate objects with 0 distance..
            translate([WALL_THICKNESS + tolerance, WALL_THICKNESS + tolerance + 6, lid_height - 0.01])
                cube([1.0, _case_main_length - WALL_THICKNESS - tolerance - 6 + WALL_THICKNESS/2, 1.5]);
            
            translate([_case_main_width - WALL_THICKNESS - tolerance - 1, WALL_THICKNESS + tolerance + 6, lid_height - 0.01])
                cube([1.0, _case_main_length - WALL_THICKNESS - tolerance - 6 + WALL_THICKNESS/2, 1.5]);
            
            translate([WALL_THICKNESS + tolerance + 6, WALL_THICKNESS + tolerance, lid_height - 0.01])
                cube([_case_main_width - 2 * WALL_THICKNESS - 2 * tolerance - 12, 1.0, 1.5]);
        }
        
        translate([-_wing_part_width + WALL_THICKNESS/2, _case_main_length + WALL_THICKNESS/2, lid_height - _tbeam_pcb_thickness + 0.01])
            tbeam();
        
        // This cube will make sure there is a bit of space for the solder-joints of the SMA connector on the older T-Beams.
        // They are quite large...
        translate([_case_main_width / 2 - 17.5, _case_total_length - WALL_THICKNESS/2 - 5 , WALL_THICKNESS])
            cube([35, 5, 2]);
        
        // screw holes
        translate([WALL_THICKNESS + 1.4, WALL_THICKNESS + 1.4])
            lid_screw_hole(); 
        translate([_case_main_width - WALL_THICKNESS - 1.4, WALL_THICKNESS + 1.4])
            lid_screw_hole(); 
        
        // Note that the T-Beam screws are _not_ symmetric.
        // On the side where the GPS chip is, distance from from side is 2.36mm (to hole-center).
        // On the other side, it is 2.67mm Thistens to the longer sides is 2.59 all around
        // Additionally, add 0.3mm tolerance
        translate([-_wing_part_width + WALL_THICKNESS / 2 + PRINTING_TOLERANCE_XY + 2.67,
                   _case_main_length + WALL_THICKNESS / 2 + 2.59 + PRINTING_TOLERANCE_XY])
            lid_screw_hole();
        translate([_case_main_width + _wing_part_width - WALL_THICKNESS / 2 - PRINTING_TOLERANCE_XY - 2.36,
                   _case_main_length + WALL_THICKNESS / 2 + 2.56 + PRINTING_TOLERANCE_XY])
            lid_screw_hole(); 
        
        
        // Fan (40mm). TODO: add 30mm variant?
        if (FAN_TYPE == "40") {
            translate([8.5 + 16, 12 + 16, -0.01])
                linear_extrude(height=WALL_THICKNESS+0.05) fan();
        } else if (FAN_TYPE == "30") {
            translate([8.5 + 16, 12 + 16, -0.01])
                linear_extrude(height=WALL_THICKNESS+0.05) fan(holedist=24, outercircle=14, innercircle=9);
        }
            
        // GPS Antenna
        translate([45, _case_total_length-_case_wing_length/2, -0.01])
            linear_extrude(height=WALL_THICKNESS + 0.05) gps_antenna_hole();
        
        // 868 and 1090 text
        translate([-_wing_part_width+8, _case_main_length + _case_wing_length / 2, -0.1]) 
            linear_extrude(height=0.3) rotate(90) scale([1, -1, 1]) text("868", valign="center", halign="center", size=8);
        
        translate([_case_main_width+8, _case_main_length + _case_wing_length / 2, -0.1]) 
            linear_extrude(height=0.3) rotate(270) scale([1, -1, 1]) text("1090", valign="center", halign="center", size=8);
    }
}


module view_side_by_side() {
    case();
    translate([-_wing_part_width - 1, _case_total_length]) rotate(180) lid();
}

module view_assembled(gap=0.1) {
    // Intersection helps to visualize if we actually have the lid and case intersect (i.e. error)
    //intersection() {
        case();
        translate([_case_main_width, 0, _case_height + gap + 4.9]) rotate([180, 0, 180]) lid();
    //}
}