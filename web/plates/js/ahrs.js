function ahrsRenderer(location_id) {
	this.width = -1;
	this.height = -1;

	this.locationID = location_id;
	this.canvas = document.getElementById(location_id);
	this.canvas_container = document.getElementById(location_id).parentElement;
	
	var ai = document.getElementById("attitude_indicator"),
	    _this = this;

	this.ps = [];
	this.rs = [];
	this.hs = [];
	this.ss = [];
	
	ai.onload = function() {
		_this.ps = ai.contentDocument.getElementsByClassName("pitch");
		_this.rs = ai.contentDocument.getElementsByClassName("roll");
		_this.hs = ai.contentDocument.getElementsByClassName("heading");
		_this.ss = ai.contentDocument.getElementsByClassName("slipSkid");
	};

}

ahrsRenderer.prototype = {
	constructor: ahrsRenderer,

	init: function () {

		this.pitch = 0;
		this.roll = 0;
		this.heading = 0;
		this.slipSkid = 0;

		this.resize();
	},

	resize: function () {
		var canvasWidth = this.canvas_container.offsetWidth - 12; // was (2*(this.canvas_container.offsetLeft)) // account for padding adjustments

		if (canvasWidth !== this.width) {
			this.width = canvasWidth;
			this.height = canvasWidth *0.5;
			
			this.canvas.width = this.width;
			this.canvas.height = this.height;
		}
	},

	orientation: function (pitch, roll, heading, slipSkid) {
	    // Assume we receive valid pitch, roll, heading
		this.pitch = pitch;
		this.roll = roll;
		this.heading = heading;
		this.slipSkid = slipSkid;
	},

	animate: function (t, pitch, roll, heading, slipSkid) {
		var FPS = 40; // we assume we can maintain a certain frame rate
		var x_inc = ((pitch - this.pitch) / (FPS * t));
		var y_inc = ((roll - this.roll) / (FPS * t));
		if ((heading < this.heading) && (this.heading - heading) > 180) {
			// let the animation wrap around gracefully clockwise
			heading += 360;
		} else if ((heading > this.heading) && (heading - this.heading) > 180) {
			// let the animation wrap around gracefully counter clockwise
			this.heading += 360;
		}
		var z_inc = ((heading - this.heading) / (FPS * t));
		var w_inc = ((slipSkid - this.slipSkid) / (FPS * t));
		var _this = this;
		var frames = 0;
		var f = function () {
			_this.pitch += x_inc;
			_this.roll += y_inc;
			_this.heading += z_inc;
			_this.slipSkid += w_inc;
			if (frames < (FPS * t)) {
				_this.draw();
				frames++;
				window.requestAnimationFrame(f); // recurse
			} else {
				_this.orientation(pitch, roll, heading, slipSkid);
			}
		};
		f();
	},

	draw: function() {
		for (i=0; i<this.ps.length; i++) {
			this.ps[i].setAttribute("transform", "translate(0,"+this.pitch * 10+")")
		}

		for (i=0; i<this.rs.length; i++) {
			this.rs[i].setAttribute("transform", "rotate("+(-this.roll)+")")
		}

		for (i=0; i<this.hs.length; i++) {
		    var h = this.heading;
		    while (h < 0) {
		    	h += 360
			}
			this.hs[i].setAttribute("transform", "translate("+(-(this.heading % 360) * 2)+",0)")
		}

		for (i=0; i<this.ss.length; i++) {
			this.ss[i].setAttribute("transform", "translate("+(-this.slipSkid * 2)+",0)")
		}
	}
};

function gMeterRenderer(locationId, plim, nlim) {
    this.plim = plim;
    this.nlim = nlim;
    this.nticks = Math.floor(plim+1) - Math.floor(nlim) + 1;

    this.width = -1;
    this.height = -1;

    this.locationId = locationId;
    this.canvas = document.getElementById(locationId);
    this.resize();

    // State variables
    this.g = 1;
    this.min = 1;
    this.max = 1;

    // Draw the G Meter using the svg.js library
    var gMeter = SVG(locationId).viewbox(-200, -200, 400, 400).group().addClass('gMeter');

    var el, card = gMeter.group().addClass('card');
    card.circle(390).cx(0).cy(0);
    card.line(-150, 0, -190, 0)
        .addClass('marks one');
    for (i=Math.floor(nlim); i<=Math.floor(plim+1); i++) {
        if (i%2 == 0) {
            el = card.line(-150, 0, -190, 0).addClass('big');
            card.text(i.toString())
                .addClass('text')
                .cx(-105).cy(0)
                .transform({ rotation: (i-1)/this.nticks*360, cx: 0, cy: 0, relative: true })
                .transform({ rotation: -(i-1)/this.nticks*360, relative: true });
        } else {
            el = card.line(-165, 0, -190, 0);

        }
        el.addClass('marks')
            .rotate((i-1)/this.nticks*360, 0, 0);
    }
    card.line(-140, 0, -190, 0).addClass('marks limit').rotate((plim-1)/this.nticks*360, 0, 0);
    card.line(-140, 0, -190, 0).addClass('marks limit').rotate((nlim-1)/this.nticks*360, 0, 0);

    var ax = -Math.cos(2*Math.PI/this.nticks),
        ay = -Math.sin(2*Math.PI/this.nticks);
    card.path('M -170 0, A 170 170 0 0 1 ' + 170*ax + ' ' + 170*ay)
        .rotate(-Math.floor(plim)/this.nticks*360, 0, 0)
        .addClass('marks')
        .style('fill-opacity', '0');
    card.path('M -175 0, A 175 175 0 0 1 ' + 175*ax + ' ' + 175*ay)
        .rotate(-Math.floor(plim)/this.nticks*360, 0, 0)
        .addClass('marks')
        .style('fill-opacity', '0');


    this.pointer_el = gMeter.group().addClass('g');
    this.pointer_el.polygon('0,0 -170,0 -150,-10 0,-10').addClass('pointer');
    this.pointer_el.polygon('0,0 -170,0 -150,+10 0,+10').addClass('pointerBG');

    this.max_el = gMeter.group().addClass('max');
    this.max_el.polygon('0,0 -170,0 -150,-5 0,-5').addClass('pointer');
    this.max_el.polygon('0,0 -170,0 -150,+5 0,+5').addClass('pointerBG');

    this.min_el = gMeter.group().addClass('min');
    this.min_el.polygon('0,0 -170,0 -160,-5 0,-5').addClass('pointer');
    this.min_el.polygon('0,0 -170,0 -160,+5 0,+5').addClass('pointerBG');

    gMeter.circle(40).cx(0).cy(0).addClass('center');

    var reset = gMeter.group().cx(-165).cy(165).addClass('reset');
    reset.circle(60).cx(0).cy(0).addClass('reset');
    reset.text("RESET").cx(0).cy(0).addClass('text');
    reset.on('click', function() {
        reset.animate(200).rotate(20, 0, 0);
        this.reset();
        reset.animate(200).rotate(0, 0, 0);
    }, this);
}

gMeterRenderer.prototype = {
    constructor: gMeterRenderer,

    resize: function () {
        var canvasWidth = this.canvas.parentElement.offsetWidth - 12;

        if (canvasWidth !== this.width) {
            this.width = canvasWidth;
            this.height = canvasWidth * 0.5;

            this.canvas.width = this.width;
            this.canvas.height = this.height;
        }
    },

    update: function (g) {
        this.g = g;
        this.max = g > this.max ? g : this.max;
        this.min = g < this.min ? g : this.min;

        this.pointer_el.animate(50).rotate((g-1)/this.nticks*360, 0, 0);
        this.max_el.animate(50).rotate((this.max-1)/this.nticks*360, 0, 0);
        this.min_el.animate(50).rotate((this.min-1)/this.nticks*360, 0, 0);
    },

    reset: function() {
        this.max = this.g;
        this.min = this.g;
    }
};
