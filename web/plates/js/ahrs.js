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

		vertex_shader = ' \
uniform mat4 u_modelViewProjMatrix; \
uniform mat4 u_normalMatrix; \
uniform vec3 lightDir; \
attribute vec3 vNormal; \
attribute vec4 vColor; \
attribute vec4 vPosition; \
varying float v_Dot; \
varying vec4 v_Color; \
void main() { \
	gl_Position = u_modelViewProjMatrix * vPosition; \
	v_Color = vColor; \
	vec4 transNormal = u_normalMatrix * vec4(vNormal, 1); \
	v_Dot = max(dot(transNormal.xyz, lightDir), 0.0); \
}';
		
		color_shader = '\
precision mediump float; \
varying float v_Dot; \
varying vec4 v_Color; \
void main() { \
	gl_FragColor = vec4(v_Color.xyz * v_Dot, v_Color.a); \
}';

		var vertexShader = loadShaderVertexScript(gl, vertex_shader);
		var colorShader = loadShaderFragmentScript(gl, color_shader);

		var program = gl.createProgram();

		gl.attachShader(program, vertexShader);
		gl.attachShader(program, colorShader);

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
		gl.uniform3f(gl.getUniformLocation(g.program, "lightDir"), 0, 1, 1); // above and back

		// Create a box. On return 'gl' contains a 'box' property with
		// the BufferObjects containing the arrays for vertices,
		// normals, texture coords, and indices.
		// g.box = makeBox(gl);
		g.box = this.makePaperAirplane(gl);

		// Create some matrices to use later and save their locations in the shaders
		g.mvMatrix = new J3DIMatrix4();
		g.u_normalMatrixLoc = gl.getUniformLocation(g.program, "u_normalMatrix");
		g.normalMatrix = new J3DIMatrix4();
		g.u_modelViewProjMatrixLoc =
			gl.getUniformLocation(g.program, "u_modelViewProjMatrix");
		g.mvpMatrix = new J3DIMatrix4();

		gl.enableVertexAttribArray(0); // lighting
		gl.enableVertexAttribArray(1); // color
		gl.enableVertexAttribArray(2); // vertices

		gl.bindBuffer(gl.ARRAY_BUFFER, g.box.vertexObject);
		gl.vertexAttribPointer(2, 3, gl.FLOAT, false, 0, 0);

		gl.bindBuffer(gl.ARRAY_BUFFER, g.box.normalObject);
		gl.vertexAttribPointer(0, 3, gl.FLOAT, false, 0, 0);
		
		gl.bindBuffer(gl.ARRAY_BUFFER, g.box.colorObject);
		gl.vertexAttribPointer(1, 4, gl.FLOAT, false, 0, 0); // was gl.UNSIGNED_BYTE
		
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
		if ((z < this.heading) && (this.heading - z) > 180) {
			// let the animation wrap aroung gracefully clockwise
			z += 360;
		} else if ((z > this.heading) && (z - this.heading) > 180) {
			// let the animation wrap aroung gracefully counter clockwise
			this.heading += 360;
		}
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

		g.mvMatrix.rotate(this.pitch, 1, 0, 0);
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
	},
	
	makePaperAirplane: function (ctx) {
		// Return an object with the following properties:
		//  normalObject        WebGLBuffer object for normals (for lighting)
		//  vertexObject        WebGLBuffer object for vertices (actual 3d object)
		//  indexObject         WebGLBuffer object for indices (index of triangles within object)
		//  numIndices          The number of indices in the indexObject (number of triangles)

		// constants to make it easy to adjust the proportions and axis of the object
		var LENGTH = 1; 	var WIDTH = 1;		var DEPTH = 0.33;		var SPREAD = 0.1;
		var CENTERX = 0;	var CENTERY = 0;	var CENTERZ = 0.5;

		// vertex coords array
		var vertices = new Float32Array([
			CENTERX, CENTERY, -LENGTH-CENTERZ,
			-WIDTH, CENTERY, LENGTH-CENTERZ,
			-SPREAD, CENTERY, LENGTH-CENTERZ, // left wing

		   	CENTERX, CENTERY, -LENGTH-CENTERZ,
			CENTERX, -DEPTH, LENGTH-CENTERZ,
			-SPREAD, CENTERY, LENGTH-CENTERZ, // left center section

		   	CENTERX, CENTERY, -LENGTH-CENTERZ,
			CENTERX, -DEPTH, LENGTH-CENTERZ,
			SPREAD, CENTERY, LENGTH-CENTERZ, // right center section

			CENTERX, CENTERY, -LENGTH-CENTERZ,
			WIDTH, CENTERY, LENGTH-CENTERZ,
			SPREAD, CENTERY, LENGTH-CENTERZ // right wing
			]);

		// normal array for light reflection and shading
		var normals = new Float32Array([
			0, 1, 0,
         	0, 1, 0,
         	0, 1, 0, // left wing actual perpendicular is up
         	1, 1, 0,
         	1, 1, 0,
         	1, 1, 0, // left center section estmated perpendicular is right
         	-1, 1, 0,
         	-1, 1, 0,
         	-1, 1, 0, // right center section estmated perpendicular is left
         	0, 1, 0,
         	0, 1, 0,
         	0, 1, 0 // right wing actual perpendicular is up
		]);


		// index array
		var indices = new Uint8Array([
			0, 1, 2, // left wing
           	3, 4, 5, // left center section
           	6, 7, 8, // right center section
           	9, 10, 11 // right wing
		]);

		// Set up the array of colors for the cube's faces
		// the tip of the paper aiplane is lighter and then a gradiant back to red for teh left and a green for the right
		var colors_rg = new Uint8Array([
			1, 0, 0, 1,
			1, 0, 0, 1,
			1, 0, 0, 1, // left wing
			   
			1, 0, 0, 1,
			1, 0, 0, 1,
			1, 0, 0, 1, // left center section
			   
			0, 1, 0, 1,
			0, 1, 0, 1,
			0, 1, 0, 1, // right center section
			   
			0, 1, 0, 1,
			0, 1, 0, 1,
			0, 1, 0, 1 // right wing
		]);

		var colors_bt = new Float32Array([
			.21, .31, .49, 1,
			.21, .31, .49, 1,
			.21, .31, .49, 1,

			.21, .31, .49, 1,
			.21, .31, .49, 1,
			.21, .31, .49, 1,
			   
			.39, .38, .22, 1,
			.39, .38, .22, 1,
			.39, .38, .22, 1,

			.39, .38, .22, 1,
			.39, .38, .22, 1,
			.39, .38, .22, 1
		]);

		var retval = {};

		retval.vertexObject = ctx.createBuffer();
		ctx.bindBuffer(ctx.ARRAY_BUFFER, retval.vertexObject);
		ctx.bufferData(ctx.ARRAY_BUFFER, vertices, ctx.STATIC_DRAW);

		retval.normalObject = ctx.createBuffer();
		ctx.bindBuffer(ctx.ARRAY_BUFFER, retval.normalObject);
		ctx.bufferData(ctx.ARRAY_BUFFER, normals, ctx.STATIC_DRAW);

		ctx.bindBuffer(ctx.ARRAY_BUFFER, null);

		retval.indexObject = ctx.createBuffer();
		ctx.bindBuffer(ctx.ELEMENT_ARRAY_BUFFER, retval.indexObject);
		ctx.bufferData(ctx.ELEMENT_ARRAY_BUFFER, indices, ctx.STATIC_DRAW);
		ctx.bindBuffer(ctx.ELEMENT_ARRAY_BUFFER, null);

		// Set up the vertex buffer for the colors
		retval.colorObject = gl.createBuffer();
		gl.bindBuffer(gl.ARRAY_BUFFER, retval.colorObject);
		gl.bufferData(gl.ARRAY_BUFFER, colors_bt, gl.STATIC_DRAW);

		retval.numIndices = indices.length;

		return retval;
	}
}