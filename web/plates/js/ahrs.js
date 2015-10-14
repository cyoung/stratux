function ahrsRenderer(location_id) {
	this.gcontext = {}; // globals
	this.gl = null;

	this.width = -1;
	this.height = -1;

	this.locationID = location_id;
	this.canvas = document.getElementById(location_id);
	this.canvas_container = document.getElementById(location_id).parentElement;
}

ahrsRenderer.prototype = {
	constructor: ahrsRenderer,

	_init_canvas: function () {
		var gl = initWebGL(this.locationID);
		if (!gl)
			return;
		this.gl = gl;

		vertex_shader = 'uniform mat4 u_modelViewProjMatrix; uniform mat4 u_normalMatrix; uniform vec3 lightDir; attribute vec3 vNormal; attribute vec4 vColor; attribute vec4 vPosition; varying float v_Dot; varying vec4 v_Color; void main() { gl_Position = u_modelViewProjMatrix * vPosition; v_Color = vColor; vec4 transNormal = u_normalMatrix * vec4(vNormal, 1); v_Dot = max(dot(transNormal.xyz, lightDir), 0.0); }';
		fragment_shader = 'precision mediump float; varying float v_Dot; varying vec4 v_Color; void main() { gl_FragColor = vec4(v_Color.xyz * v_Dot, v_Color.a * 0.95); }';

		var vertexShader = loadShaderVertexScript(gl, vertex_shader);
		var fragmentShader = loadShaderFragmentScript(gl, fragment_shader);

		var program = gl.createProgram();

		gl.attachShader(program, vertexShader);
		gl.attachShader(program, fragmentShader);

		// Bind attributes
		gl.bindAttribLocation(program, 0, "vNormal");
		gl.bindAttribLocation(program, 1, "vColor");
		gl.bindAttribLocation(program, 2, "vPosition");

		gl.linkProgram(program);

		var linked = gl.getProgramParameter(program, gl.LINK_STATUS);
		if (!linked && !gl.isContextLost()) {
			// something went wrong with the link
			var error = gl.getProgramInfoLog(program);
			log("Error in program linking:" + error);
			gl.deleteProgram(program);
			gl.deleteProgram(fragmentShader);
			gl.deleteProgram(vertexShader);
		} else {
			gl.useProgram(program);
			gl.clearColor(0, 0, 0, 0) // rgba for background color
			gl.clearDepth(10000); //??

			if (false /* funcky blending */ ) {
				gl.disable(gl.DEPTH_TEST);
				gl.enable(gl.BLEND);
				gl.depthFunc(gl.LESS);
				gl.blendFunc(gl.SRC_ALPHA, gl.ONE);
			} else {
				gl.enable(gl.DEPTH_TEST);
				gl.enable(gl.BLEND);
				gl.blendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA);
			}

			this.gcontext.program = program;
		}
	},

	init: function () {
		this._init_canvas(); // init the canvas
		g = this.gcontext;
		gl = this.gl;

		if (gl === null)
			return;

		// Set up a uniform variable for the shaders
		gl.uniform3f(gl.getUniformLocation(g.program, "lightDir"), 0, 1, -1);

		// Create a box. On return 'gl' contains a 'box' property with
		// the BufferObjects containing the arrays for vertices,
		// normals, texture coords, and indices.
		// g.box = makeBox(gl);
		g.box = makePaperAirplane(gl);

		// Create some matrices to use later and save their locations in the shaders
		g.mvMatrix = new J3DIMatrix4();
		g.u_normalMatrixLoc = gl.getUniformLocation(g.program, "u_normalMatrix");
		g.normalMatrix = new J3DIMatrix4();
		g.u_modelViewProjMatrixLoc =
			gl.getUniformLocation(g.program, "u_modelViewProjMatrix");
		g.mvpMatrix = new J3DIMatrix4();

		// Enable all of the vertex attribute arrays.
		gl.enableVertexAttribArray(0);
		gl.enableVertexAttribArray(1);
		gl.enableVertexAttribArray(2);
		// Set up all the vertex attributes for vertices, normals and colors
		gl.bindBuffer(gl.ARRAY_BUFFER, g.box.vertexObject);
		gl.vertexAttribPointer(2, 3, gl.FLOAT, false, 0, 0);
		gl.bindBuffer(gl.ARRAY_BUFFER, g.box.normalObject);
		gl.vertexAttribPointer(0, 3, gl.FLOAT, false, 0, 0);
		gl.bindBuffer(gl.ARRAY_BUFFER, g.box.colorObject);
		gl.vertexAttribPointer(1, 4, gl.UNSIGNED_BYTE, false, 0, 0);

		// Bind the index array
		gl.bindBuffer(gl.ELEMENT_ARRAY_BUFFER, g.box.indexObject);

		this.pitch = 0;
		this.roll = 0;
		this.heading = 0;

		this.resize();
	},

	resize: function () {
		gl = this.gl;
		g = this.gcontext;
		if (gl === null)
			return;

		var canvasWidth = this.canvas_container.offsetWidth - 12; // was (2*(this.canvas_container.offsetLeft)) // account for padding adjustments

		if (canvasWidth !== this.width) {
			this.width = canvasWidth;
			this.height = canvasWidth *0.5;
			
			this.canvas.width = this.width;
			this.canvas.height = this.height;
			// Set the viewport and projection matrix for the scene
			gl.viewport(0, 0, this.width, this.height);
			g.perspectiveMatrix = new J3DIMatrix4();
			g.perspectiveMatrix.perspective(30, this.width / this.height, 1, 10000);
			g.perspectiveMatrix.lookat(	0, 0, 4, // eye location
										0, 0, 0, // focal point
										0, 1, 0); // up vector
		}
	},

	orientation: function (x, y, z) {
		if (x > 360) x -= 360;
		if (y > 360) y -= 360;
		if (z > 360) z -= 360;
		this.pitch = x; // need to reorient to level 
		this.roll = y;
		this.heading = z;
	},

	animate: function (t, x, y, z) {
		var FPS = 24; // we assume we can maintain a certain frame rate
		var x_inc = ((x - this.pitch) / (FPS * t));
		var y_inc = ((y - this.roll) / (FPS * t));
		// let the animation wrap aroung gracefully
		if ((z < this.heading) && (this.heading - z) > 180)
			z += 360;
		var z_inc = ((z - this.heading) / (FPS * t));
		var _this = this;
		//console.log(z_inc);
		var frames = 0;
		var f = function () {
			_this.pitch += x_inc;
			_this.roll += y_inc;
			_this.heading += z_inc;
			if (frames < (FPS * t)) {
				_this.draw();
				frames++
				window.requestAnimationFrame(f, _this.canvas); // recurse
			} else {
				_this.orientation(x, y, z);
			}
		};
		f();
	},

	draw: function () {
		this.resize();
		gl = this.gl;
		g = this.gcontext;
		if (gl === null)
			return;

		// Clear the canvas
		gl.clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT);

		// Make a model/view matrix.
		g.mvMatrix.makeIdentity();

		g.mvMatrix.rotate(15, 1, 0, 0); // adjust viewing angle slightly by pitching the airplane up

		g.mvMatrix.rotate(-this.pitch, 1, 0, 0);
		g.mvMatrix.rotate(-this.roll, 0, 0, 1);
		g.mvMatrix.rotate(-this.heading, 0, 1, 0);

		// Construct the normal matrix from the model-view matrix and pass it in
		g.normalMatrix.load(g.mvMatrix);
		g.normalMatrix.invert();
		g.normalMatrix.transpose();
		g.normalMatrix.setUniform(gl, g.u_normalMatrixLoc, false);

		// Construct the model-view * projection matrix and pass it in
		g.mvpMatrix.load(g.perspectiveMatrix);
		g.mvpMatrix.multiply(g.mvMatrix);
		g.mvpMatrix.setUniform(gl, g.u_modelViewProjMatrixLoc, false);

		// Draw the object
		gl.drawElements(gl.TRIANGLES, g.box.numIndices, gl.UNSIGNED_BYTE, 0);
	}
}