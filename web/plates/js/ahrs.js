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

	orientation: function (pitch, roll, heading) {
	    // Assume we receive valid pitch, roll, heading
		this.pitch = pitch;
		this.roll = roll;
		this.heading = heading;
	},

	animate: function (t, pitch, roll, heading) {
		var FPS = 100; // we assume we can maintain a certain frame rate
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
		var _this = this;
		//console.log(z_inc);
		var frames = 0;
		var f = function () {
			_this.pitch += x_inc;
			_this.roll += y_inc;
			_this.heading += z_inc;
			if (frames < (FPS * t)) {
				_this.draw();
				frames++;
				window.requestAnimationFrame(f); // recurse
			} else {
				_this.orientation(pitch, roll, heading);
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
