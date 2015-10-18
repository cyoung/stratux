/*
 * Copyright 2010, Google Inc.
 * All rights reserved.
 *
 * Redistribution and use in source and binary forms, with or without
 * modification, are permitted provided that the following conditions are
 * met:
 *
 *     * Redistributions of source code must retain the above copyright
 * notice, this list of conditions and the following disclaimer.
 *     * Redistributions in binary form must reproduce the above
 * copyright notice, this list of conditions and the following disclaimer
 * in the documentation and/or other materials provided with the
 * distribution.
 *     * Neither the name of Google Inc. nor the names of its
 * contributors may be used to endorse or promote products derived from
 * this software without specific prior written permission.
 *
 * THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
 * "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
 * LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
 * A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
 * OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
 * SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
 * LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
 * DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
 * THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
 * (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
 * OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
 */


/**
 * @fileoverview This file contains functions every webgl program will need
 * a version of one way or another.
 *
 * Instead of setting up a context manually it is recommended to
 * use. This will check for success or failure. On failure it
 * will attempt to present an approriate message to the user.
 *
 *       gl = WebGLUtils.setupWebGL(canvas);
 *
 * For animated WebGL apps use of setTimeout or setInterval are
 * discouraged. It is recommended you structure your rendering
 * loop like this.
 *
 *       function render() {
 *         window.requestAnimFrame(render, canvas);
 *
 *         // do rendering
 *         ...
 *       }
 *       render();
 *
 * This will call your rendering function up to the refresh rate
 * of your display but will stop rendering if your app is not
 * visible.
 */

WebGLUtils = function () {

	/**
	 * Creates the HTLM for a failure message
	 * @param {string} canvasContainerId id of container of th
	 *        canvas.
	 * @return {string} The html.
	 */
	var makeFailHTML = function (msg) {
		return '' +
			'<table style="background-color: #8CE; width: 100%; height: 100%;"><tr>' +
			'<td align="center">' +
			'<div style="display: table-cell; vertical-align: middle;">' +
			'<div style="">' + msg + '</div>' +
			'</div>' +
			'</td></tr></table>';
	};

	/**
	 * Mesasge for getting a webgl browser
	 * @type {string}
	 */
	var GET_A_WEBGL_BROWSER = '' +
		'This page requires a browser that supports WebGL.<br/>' +
		'<a href="http://get.webgl.org">Click here to upgrade your browser.</a>';

	/**
	 * Mesasge for need better hardware
	 * @type {string}
	 */
	var OTHER_PROBLEM = '' +
		"It doesn't appear your computer can support WebGL.<br/>" +
		'<a href="http://get.webgl.org/troubleshooting/">Click here for more information.</a>';

	/**
	 * Creates a webgl context. If creation fails it will
	 * change the contents of the container of the <canvas>
	 * tag to an error message with the correct links for WebGL.
	 * @param {Element} canvas. The canvas element to create a
	 *     context from.
	 * @param {WebGLContextCreationAttirbutes} opt_attribs Any
	 *     creation attributes you want to pass in.
	 * @return {WebGLRenderingContext} The created context.
	 */
	var setupWebGL = function (canvas, opt_attribs) {
		function showLink(str) {
			var container = canvas.parentNode;
			if (container) {
				container.innerHTML = makeFailHTML(str);
			}
		};

		if (!window.WebGLRenderingContext) {
			showLink(GET_A_WEBGL_BROWSER);
			return null;
		}

		var context = create3DContext(canvas, opt_attribs);
		if (!context) {
			showLink(OTHER_PROBLEM);
		}
		return context;
	};

	/**
	 * Creates a webgl context.
	 * @param {!Canvas} canvas The canvas tag to get context
	 *     from. If one is not passed in one will be created.
	 * @return {!WebGLContext} The created context.
	 */
	var create3DContext = function (canvas, opt_attribs) {
		var names = ["webgl", "experimental-webgl", "webkit-3d", "moz-webgl"];
		var context = null;
		for (var ii = 0; ii < names.length; ++ii) {
			try {
				context = canvas.getContext(names[ii], opt_attribs);
			} catch (e) {}
			if (context) {
				break;
			}
		}
		return context;
	};

	return {
		create3DContext: create3DContext,
		setupWebGL: setupWebGL
	};
}();

/**
 * Provides requestAnimationFrame in a cross browser way.
 */
window.requestAnimFrame = (function () {
	return window.requestAnimationFrame ||
		window.webkitRequestAnimationFrame ||
		window.mozRequestAnimationFrame ||
		window.oRequestAnimationFrame ||
		window.msRequestAnimationFrame ||
		function ( /* function FrameRequestCallback */ callback, /* DOMElement Element */ element) {
			return window.setTimeout(callback, 1000 / 60);
		};
})();

/**
 * Provides cancelAnimationFrame in a cross browser way.
 */
window.cancelAnimFrame = (function () {
	return window.cancelAnimationFrame ||
		window.webkitCancelAnimationFrame ||
		window.mozCancelAnimationFrame ||
		window.oCancelAnimationFrame ||
		window.msCancelAnimationFrame ||
		window.clearTimeout;
})();


/*
 ** Copyright (c) 2012 The Khronos Group Inc.
 **
 ** Permission is hereby granted, free of charge, to any person obtaining a
 ** copy of this software and/or associated documentation files (the
 ** "Materials"), to deal in the Materials without restriction, including
 ** without limitation the rights to use, copy, modify, merge, publish,
 ** distribute, sublicense, and/or sell copies of the Materials, and to
 ** permit persons to whom the Materials are furnished to do so, subject to
 ** the following conditions:
 **
 ** The above copyright notice and this permission notice shall be included
 ** in all copies or substantial portions of the Materials.
 **
 ** THE MATERIALS ARE PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
 ** EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
 ** MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
 ** IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY
 ** CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
 ** TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE
 ** MATERIALS OR THE USE OR OTHER DEALINGS IN THE MATERIALS.
 */

// Various functions for helping debug WebGL apps.

WebGLDebugUtils = function () {

	/**
	 * Wrapped logging function.
	 * @param {string} msg Message to log.
	 */
	var log = function (msg) {
		if (window.console && window.console.log) {
			window.console.log(msg);
		}
	};

	/**
	 * Wrapped error logging function.
	 * @param {string} msg Message to log.
	 */
	var error = function (msg) {
		if (window.console && window.console.error) {
			window.console.error(msg);
		} else {
			log(msg);
		}
	};


	/**
	 * Which arguments are enums based on the number of arguments to the function.
	 * So
	 *    'texImage2D': {
	 *       9: { 0:true, 2:true, 6:true, 7:true },
	 *       6: { 0:true, 2:true, 3:true, 4:true },
	 *    },
	 *
	 * means if there are 9 arguments then 6 and 7 are enums, if there are 6
	 * arguments 3 and 4 are enums
	 *
	 * @type {!Object.<number, !Object.<number, string>}
	 */
	var glValidEnumContexts = {
		// Generic setters and getters

		'enable': {
			1: {
				0: true
			}
		},
		'disable': {
			1: {
				0: true
			}
		},
		'getParameter': {
			1: {
				0: true
			}
		},

		// Rendering

		'drawArrays': {
			3: {
				0: true
			}
		},
		'drawElements': {
			4: {
				0: true,
				2: true
			}
		},

		// Shaders

		'createShader': {
			1: {
				0: true
			}
		},
		'getShaderParameter': {
			2: {
				1: true
			}
		},
		'getProgramParameter': {
			2: {
				1: true
			}
		},
		'getShaderPrecisionFormat': {
			2: {
				0: true,
				1: true
			}
		},

		// Vertex attributes

		'getVertexAttrib': {
			2: {
				1: true
			}
		},
		'vertexAttribPointer': {
			6: {
				2: true
			}
		},

		// Textures

		'bindTexture': {
			2: {
				0: true
			}
		},
		'activeTexture': {
			1: {
				0: true
			}
		},
		'getTexParameter': {
			2: {
				0: true,
				1: true
			}
		},
		'texParameterf': {
			3: {
				0: true,
				1: true
			}
		},
		'texParameteri': {
			3: {
				0: true,
				1: true,
				2: true
			}
		},
		'texImage2D': {
			9: {
				0: true,
				2: true,
				6: true,
				7: true
			},
			6: {
				0: true,
				2: true,
				3: true,
				4: true
			}
		},
		'texSubImage2D': {
			9: {
				0: true,
				6: true,
				7: true
			},
			7: {
				0: true,
				4: true,
				5: true
			}
		},
		'copyTexImage2D': {
			8: {
				0: true,
				2: true
			}
		},
		'copyTexSubImage2D': {
			8: {
				0: true
			}
		},
		'generateMipmap': {
			1: {
				0: true
			}
		},
		'compressedTexImage2D': {
			7: {
				0: true,
				2: true
			}
		},
		'compressedTexSubImage2D': {
			8: {
				0: true,
				6: true
			}
		},

		// Buffer objects

		'bindBuffer': {
			2: {
				0: true
			}
		},
		'bufferData': {
			3: {
				0: true,
				2: true
			}
		},
		'bufferSubData': {
			3: {
				0: true
			}
		},
		'getBufferParameter': {
			2: {
				0: true,
				1: true
			}
		},

		// Renderbuffers and framebuffers

		'pixelStorei': {
			2: {
				0: true,
				1: true
			}
		},
		'readPixels': {
			7: {
				4: true,
				5: true
			}
		},
		'bindRenderbuffer': {
			2: {
				0: true
			}
		},
		'bindFramebuffer': {
			2: {
				0: true
			}
		},
		'checkFramebufferStatus': {
			1: {
				0: true
			}
		},
		'framebufferRenderbuffer': {
			4: {
				0: true,
				1: true,
				2: true
			}
		},
		'framebufferTexture2D': {
			5: {
				0: true,
				1: true,
				2: true
			}
		},
		'getFramebufferAttachmentParameter': {
			3: {
				0: true,
				1: true,
				2: true
			}
		},
		'getRenderbufferParameter': {
			2: {
				0: true,
				1: true
			}
		},
		'renderbufferStorage': {
			4: {
				0: true,
				1: true
			}
		},

		// Frame buffer operations (clear, blend, depth test, stencil)

		'clear': {
			1: {
				0: {
					'enumBitwiseOr': ['COLOR_BUFFER_BIT', 'DEPTH_BUFFER_BIT', 'STENCIL_BUFFER_BIT']
				}
			}
		},
		'depthFunc': {
			1: {
				0: true
			}
		},
		'blendFunc': {
			2: {
				0: true,
				1: true
			}
		},
		'blendFuncSeparate': {
			4: {
				0: true,
				1: true,
				2: true,
				3: true
			}
		},
		'blendEquation': {
			1: {
				0: true
			}
		},
		'blendEquationSeparate': {
			2: {
				0: true,
				1: true
			}
		},
		'stencilFunc': {
			3: {
				0: true
			}
		},
		'stencilFuncSeparate': {
			4: {
				0: true,
				1: true
			}
		},
		'stencilMaskSeparate': {
			2: {
				0: true
			}
		},
		'stencilOp': {
			3: {
				0: true,
				1: true,
				2: true
			}
		},
		'stencilOpSeparate': {
			4: {
				0: true,
				1: true,
				2: true,
				3: true
			}
		},

		// Culling

		'cullFace': {
			1: {
				0: true
			}
		},
		'frontFace': {
			1: {
				0: true
			}
		},

		// ANGLE_instanced_arrays extension

		'drawArraysInstancedANGLE': {
			4: {
				0: true
			}
		},
		'drawElementsInstancedANGLE': {
			5: {
				0: true,
				2: true
			}
		},

		// EXT_blend_minmax extension

		'blendEquationEXT': {
			1: {
				0: true
			}
		}
	};

	/**
	 * Map of numbers to names.
	 * @type {Object}
	 */
	var glEnums = null;

	/**
	 * Map of names to numbers.
	 * @type {Object}
	 */
	var enumStringToValue = null;

	/**
	 * Initializes this module. Safe to call more than once.
	 * @param {!WebGLRenderingContext} ctx A WebGL context. If
	 *    you have more than one context it doesn't matter which one
	 *    you pass in, it is only used to pull out constants.
	 */
	function init(ctx) {
		if (glEnums == null) {
			glEnums = {};
			enumStringToValue = {};
			for (var propertyName in ctx) {
				if (typeof ctx[propertyName] == 'number') {
					glEnums[ctx[propertyName]] = propertyName;
					enumStringToValue[propertyName] = ctx[propertyName];
				}
			}
		}
	}

	/**
	 * Checks the utils have been initialized.
	 */
	function checkInit() {
		if (glEnums == null) {
			throw 'WebGLDebugUtils.init(ctx) not called';
		}
	}

	/**
	 * Returns true or false if value matches any WebGL enum
	 * @param {*} value Value to check if it might be an enum.
	 * @return {boolean} True if value matches one of the WebGL defined enums
	 */
	function mightBeEnum(value) {
		checkInit();
		return (glEnums[value] !== undefined);
	}

	/**
	 * Gets an string version of an WebGL enum.
	 *
	 * Example:
	 *   var str = WebGLDebugUtil.glEnumToString(ctx.getError());
	 *
	 * @param {number} value Value to return an enum for
	 * @return {string} The string version of the enum.
	 */
	function glEnumToString(value) {
		checkInit();
		var name = glEnums[value];
		return (name !== undefined) ? ("gl." + name) :
			("/*UNKNOWN WebGL ENUM*/ 0x" + value.toString(16) + "");
	}

	/**
	 * Returns the string version of a WebGL argument.
	 * Attempts to convert enum arguments to strings.
	 * @param {string} functionName the name of the WebGL function.
	 * @param {number} numArgs the number of arguments passed to the function.
	 * @param {number} argumentIndx the index of the argument.
	 * @param {*} value The value of the argument.
	 * @return {string} The value as a string.
	 */
	function glFunctionArgToString(functionName, numArgs, argumentIndex, value) {
		var funcInfo = glValidEnumContexts[functionName];
		if (funcInfo !== undefined) {
			var funcInfo = funcInfo[numArgs];
			if (funcInfo !== undefined) {
				if (funcInfo[argumentIndex]) {
					if (typeof funcInfo[argumentIndex] === 'object' &&
						funcInfo[argumentIndex]['enumBitwiseOr'] !== undefined) {
						var enums = funcInfo[argumentIndex]['enumBitwiseOr'];
						var orResult = 0;
						var orEnums = [];
						for (var i = 0; i < enums.length; ++i) {
							var enumValue = enumStringToValue[enums[i]];
							if ((value & enumValue) !== 0) {
								orResult |= enumValue;
								orEnums.push(glEnumToString(enumValue));
							}
						}
						if (orResult === value) {
							return orEnums.join(' | ');
						} else {
							return glEnumToString(value);
						}
					} else {
						return glEnumToString(value);
					}
				}
			}
		}
		if (value === null) {
			return "null";
		} else if (value === undefined) {
			return "undefined";
		} else {
			return value.toString();
		}
	}

	/**
	 * Converts the arguments of a WebGL function to a string.
	 * Attempts to convert enum arguments to strings.
	 *
	 * @param {string} functionName the name of the WebGL function.
	 * @param {number} args The arguments.
	 * @return {string} The arguments as a string.
	 */
	function glFunctionArgsToString(functionName, args) {
		// apparently we can't do args.join(",");
		var argStr = "";
		var numArgs = args.length;
		for (var ii = 0; ii < numArgs; ++ii) {
			argStr += ((ii == 0) ? '' : ', ') +
				glFunctionArgToString(functionName, numArgs, ii, args[ii]);
		}
		return argStr;
	};


	function makePropertyWrapper(wrapper, original, propertyName) {
		//log("wrap prop: " + propertyName);
		wrapper.__defineGetter__(propertyName, function () {
			return original[propertyName];
		});
		// TODO(gmane): this needs to handle properties that take more than
		// one value?
		wrapper.__defineSetter__(propertyName, function (value) {
			//log("set: " + propertyName);
			original[propertyName] = value;
		});
	}

	// Makes a function that calls a function on another object.
	function makeFunctionWrapper(original, functionName) {
		//log("wrap fn: " + functionName);
		var f = original[functionName];
		return function () {
			//log("call: " + functionName);
			var result = f.apply(original, arguments);
			return result;
		};
	}

	/**
	 * Given a WebGL context returns a wrapped context that calls
	 * gl.getError after every command and calls a function if the
	 * result is not gl.NO_ERROR.
	 *
	 * @param {!WebGLRenderingContext} ctx The webgl context to
	 *        wrap.
	 * @param {!function(err, funcName, args): void} opt_onErrorFunc
	 *        The function to call when gl.getError returns an
	 *        error. If not specified the default function calls
	 *        console.log with a message.
	 * @param {!function(funcName, args): void} opt_onFunc The
	 *        function to call when each webgl function is called.
	 *        You can use this to log all calls for example.
	 * @param {!WebGLRenderingContext} opt_err_ctx The webgl context
	 *        to call getError on if different than ctx.
	 */
	function makeDebugContext(ctx, opt_onErrorFunc, opt_onFunc, opt_err_ctx) {
		opt_err_ctx = opt_err_ctx || ctx;
		init(ctx);
		opt_onErrorFunc = opt_onErrorFunc || function (err, functionName, args) {
			// apparently we can't do args.join(",");
			var argStr = "";
			var numArgs = args.length;
			for (var ii = 0; ii < numArgs; ++ii) {
				argStr += ((ii == 0) ? '' : ', ') +
					glFunctionArgToString(functionName, numArgs, ii, args[ii]);
			}
			error("WebGL error " + glEnumToString(err) + " in " + functionName +
				"(" + argStr + ")");
		};

		// Holds booleans for each GL error so after we get the error ourselves
		// we can still return it to the client app.
		var glErrorShadow = {};

		// Makes a function that calls a WebGL function and then calls getError.
		function makeErrorWrapper(ctx, functionName) {
			return function () {
				if (opt_onFunc) {
					opt_onFunc(functionName, arguments);
				}
				var result = ctx[functionName].apply(ctx, arguments);
				var err = opt_err_ctx.getError();
				if (err != 0) {
					glErrorShadow[err] = true;
					opt_onErrorFunc(err, functionName, arguments);
				}
				return result;
			};
		}

		// Make a an object that has a copy of every property of the WebGL context
		// but wraps all functions.
		var wrapper = {};
		for (var propertyName in ctx) {
			if (typeof ctx[propertyName] == 'function') {
				if (propertyName != 'getExtension') {
					wrapper[propertyName] = makeErrorWrapper(ctx, propertyName);
				} else {
					var wrapped = makeErrorWrapper(ctx, propertyName);
					wrapper[propertyName] = function () {
						var result = wrapped.apply(ctx, arguments);
						return makeDebugContext(result, opt_onErrorFunc, opt_onFunc, opt_err_ctx);
					};
				}
			} else {
				makePropertyWrapper(wrapper, ctx, propertyName);
			}
		}

		// Override the getError function with one that returns our saved results.
		wrapper.getError = function () {
			for (var err in glErrorShadow) {
				if (glErrorShadow.hasOwnProperty(err)) {
					if (glErrorShadow[err]) {
						glErrorShadow[err] = false;
						return err;
					}
				}
			}
			return ctx.NO_ERROR;
		};

		return wrapper;
	}

	function resetToInitialState(ctx) {
		var numAttribs = ctx.getParameter(ctx.MAX_VERTEX_ATTRIBS);
		var tmp = ctx.createBuffer();
		ctx.bindBuffer(ctx.ARRAY_BUFFER, tmp);
		for (var ii = 0; ii < numAttribs; ++ii) {
			ctx.disableVertexAttribArray(ii);
			ctx.vertexAttribPointer(ii, 4, ctx.FLOAT, false, 0, 0);
			ctx.vertexAttrib1f(ii, 0);
		}
		ctx.deleteBuffer(tmp);

		var numTextureUnits = ctx.getParameter(ctx.MAX_TEXTURE_IMAGE_UNITS);
		for (var ii = 0; ii < numTextureUnits; ++ii) {
			ctx.activeTexture(ctx.TEXTURE0 + ii);
			ctx.bindTexture(ctx.TEXTURE_CUBE_MAP, null);
			ctx.bindTexture(ctx.TEXTURE_2D, null);
		}

		ctx.activeTexture(ctx.TEXTURE0);
		ctx.useProgram(null);
		ctx.bindBuffer(ctx.ARRAY_BUFFER, null);
		ctx.bindBuffer(ctx.ELEMENT_ARRAY_BUFFER, null);
		ctx.bindFramebuffer(ctx.FRAMEBUFFER, null);
		ctx.bindRenderbuffer(ctx.RENDERBUFFER, null);
		ctx.disable(ctx.BLEND);
		ctx.disable(ctx.CULL_FACE);
		ctx.disable(ctx.DEPTH_TEST);
		ctx.disable(ctx.DITHER);
		ctx.disable(ctx.SCISSOR_TEST);
		ctx.blendColor(0, 0, 0, 0);
		ctx.blendEquation(ctx.FUNC_ADD);
		ctx.blendFunc(ctx.ONE, ctx.ZERO);
		ctx.clearColor(0, 0, 0, 0);
		ctx.clearDepth(1);
		ctx.clearStencil(-1);
		ctx.colorMask(true, true, true, true);
		ctx.cullFace(ctx.BACK);
		ctx.depthFunc(ctx.LESS);
		ctx.depthMask(true);
		ctx.depthRange(0, 1);
		ctx.frontFace(ctx.CCW);
		ctx.hint(ctx.GENERATE_MIPMAP_HINT, ctx.DONT_CARE);
		ctx.lineWidth(1);
		ctx.pixelStorei(ctx.PACK_ALIGNMENT, 4);
		ctx.pixelStorei(ctx.UNPACK_ALIGNMENT, 4);
		ctx.pixelStorei(ctx.UNPACK_FLIP_Y_WEBGL, false);
		ctx.pixelStorei(ctx.UNPACK_PREMULTIPLY_ALPHA_WEBGL, false);
		// TODO: Delete this IF.
		if (ctx.UNPACK_COLORSPACE_CONVERSION_WEBGL) {
			ctx.pixelStorei(ctx.UNPACK_COLORSPACE_CONVERSION_WEBGL, ctx.BROWSER_DEFAULT_WEBGL);
		}
		ctx.polygonOffset(0, 0);
		ctx.sampleCoverage(1, false);
		ctx.scissor(0, 0, ctx.canvas.width, ctx.canvas.height);
		ctx.stencilFunc(ctx.ALWAYS, 0, 0xFFFFFFFF);
		ctx.stencilMask(0xFFFFFFFF);
		ctx.stencilOp(ctx.KEEP, ctx.KEEP, ctx.KEEP);
		ctx.viewport(0, 0, ctx.canvas.width, ctx.canvas.height);
		ctx.clear(ctx.COLOR_BUFFER_BIT | ctx.DEPTH_BUFFER_BIT | ctx.STENCIL_BUFFER_BIT);

		// TODO: This should NOT be needed but Firefox fails with 'hint'
		while (ctx.getError());
	}

	function makeLostContextSimulatingCanvas(canvas) {
		var unwrappedContext_;
		var wrappedContext_;
		var onLost_ = [];
		var onRestored_ = [];
		var wrappedContext_ = {};
		var contextId_ = 1;
		var contextLost_ = false;
		var resourceId_ = 0;
		var resourceDb_ = [];
		var numCallsToLoseContext_ = 0;
		var numCalls_ = 0;
		var canRestore_ = false;
		var restoreTimeout_ = 0;

		// Holds booleans for each GL error so can simulate errors.
		var glErrorShadow_ = {};

		canvas.getContext = function (f) {
			return function () {
				var ctx = f.apply(canvas, arguments);
				// Did we get a context and is it a WebGL context?
				if (ctx instanceof WebGLRenderingContext) {
					if (ctx != unwrappedContext_) {
						if (unwrappedContext_) {
							throw "got different context"
						}
						unwrappedContext_ = ctx;
						wrappedContext_ = makeLostContextSimulatingContext(unwrappedContext_);
					}
					return wrappedContext_;
				}
				return ctx;
			}
		}(canvas.getContext);

		function wrapEvent(listener) {
			if (typeof (listener) == "function") {
				return listener;
			} else {
				return function (info) {
					listener.handleEvent(info);
				}
			}
		}

		var addOnContextLostListener = function (listener) {
			onLost_.push(wrapEvent(listener));
		};

		var addOnContextRestoredListener = function (listener) {
			onRestored_.push(wrapEvent(listener));
		};


		function wrapAddEventListener(canvas) {
			var f = canvas.addEventListener;
			canvas.addEventListener = function (type, listener, bubble) {
				switch (type) {
				case 'webglcontextlost':
					addOnContextLostListener(listener);
					break;
				case 'webglcontextrestored':
					addOnContextRestoredListener(listener);
					break;
				default:
					f.apply(canvas, arguments);
				}
			};
		}

		wrapAddEventListener(canvas);

		canvas.loseContext = function () {
			if (!contextLost_) {
				contextLost_ = true;
				numCallsToLoseContext_ = 0;
				++contextId_;
				while (unwrappedContext_.getError());
				clearErrors();
				glErrorShadow_[unwrappedContext_.CONTEXT_LOST_WEBGL] = true;
				var event = makeWebGLContextEvent("context lost");
				var callbacks = onLost_.slice();
				setTimeout(function () {
					//log("numCallbacks:" + callbacks.length);
					for (var ii = 0; ii < callbacks.length; ++ii) {
						//log("calling callback:" + ii);
						callbacks[ii](event);
					}
					if (restoreTimeout_ >= 0) {
						setTimeout(function () {
							canvas.restoreContext();
						}, restoreTimeout_);
					}
				}, 0);
			}
		};

		canvas.restoreContext = function () {
			if (contextLost_) {
				if (onRestored_.length) {
					setTimeout(function () {
						if (!canRestore_) {
							throw "can not restore. webglcontestlost listener did not call event.preventDefault";
						}
						freeResources();
						resetToInitialState(unwrappedContext_);
						contextLost_ = false;
						numCalls_ = 0;
						canRestore_ = false;
						var callbacks = onRestored_.slice();
						var event = makeWebGLContextEvent("context restored");
						for (var ii = 0; ii < callbacks.length; ++ii) {
							callbacks[ii](event);
						}
					}, 0);
				}
			}
		};

		canvas.loseContextInNCalls = function (numCalls) {
			if (contextLost_) {
				throw "You can not ask a lost contet to be lost";
			}
			numCallsToLoseContext_ = numCalls_ + numCalls;
		};

		canvas.getNumCalls = function () {
			return numCalls_;
		};

		canvas.setRestoreTimeout = function (timeout) {
			restoreTimeout_ = timeout;
		};

		function isWebGLObject(obj) {
			//return false;
			return (obj instanceof WebGLBuffer ||
				obj instanceof WebGLFramebuffer ||
				obj instanceof WebGLProgram ||
				obj instanceof WebGLRenderbuffer ||
				obj instanceof WebGLShader ||
				obj instanceof WebGLTexture);
		}

		function checkResources(args) {
			for (var ii = 0; ii < args.length; ++ii) {
				var arg = args[ii];
				if (isWebGLObject(arg)) {
					return arg.__webglDebugContextLostId__ == contextId_;
				}
			}
			return true;
		}

		function clearErrors() {
			var k = Object.keys(glErrorShadow_);
			for (var ii = 0; ii < k.length; ++ii) {
				delete glErrorShadow_[k];
			}
		}

		function loseContextIfTime() {
			++numCalls_;
			if (!contextLost_) {
				if (numCallsToLoseContext_ == numCalls_) {
					canvas.loseContext();
				}
			}
		}

		// Makes a function that simulates WebGL when out of context.
		function makeLostContextFunctionWrapper(ctx, functionName) {
			var f = ctx[functionName];
			return function () {
				// log("calling:" + functionName);
				// Only call the functions if the context is not lost.
				loseContextIfTime();
				if (!contextLost_) {
					//if (!checkResources(arguments)) {
					//  glErrorShadow_[wrappedContext_.INVALID_OPERATION] = true;
					//  return;
					//}
					var result = f.apply(ctx, arguments);
					return result;
				}
			};
		}

		function freeResources() {
			for (var ii = 0; ii < resourceDb_.length; ++ii) {
				var resource = resourceDb_[ii];
				if (resource instanceof WebGLBuffer) {
					unwrappedContext_.deleteBuffer(resource);
				} else if (resource instanceof WebGLFramebuffer) {
					unwrappedContext_.deleteFramebuffer(resource);
				} else if (resource instanceof WebGLProgram) {
					unwrappedContext_.deleteProgram(resource);
				} else if (resource instanceof WebGLRenderbuffer) {
					unwrappedContext_.deleteRenderbuffer(resource);
				} else if (resource instanceof WebGLShader) {
					unwrappedContext_.deleteShader(resource);
				} else if (resource instanceof WebGLTexture) {
					unwrappedContext_.deleteTexture(resource);
				}
			}
		}

		function makeWebGLContextEvent(statusMessage) {
			return {
				statusMessage: statusMessage,
				preventDefault: function () {
					canRestore_ = true;
				}
			};
		}

		return canvas;

		function makeLostContextSimulatingContext(ctx) {
			// copy all functions and properties to wrapper
			for (var propertyName in ctx) {
				if (typeof ctx[propertyName] == 'function') {
					wrappedContext_[propertyName] = makeLostContextFunctionWrapper(
						ctx, propertyName);
				} else {
					makePropertyWrapper(wrappedContext_, ctx, propertyName);
				}
			}

			// Wrap a few functions specially.
			wrappedContext_.getError = function () {
				loseContextIfTime();
				if (!contextLost_) {
					var err;
					while (err = unwrappedContext_.getError()) {
						glErrorShadow_[err] = true;
					}
				}
				for (var err in glErrorShadow_) {
					if (glErrorShadow_[err]) {
						delete glErrorShadow_[err];
						return err;
					}
				}
				return wrappedContext_.NO_ERROR;
			};

			var creationFunctions = [
      "createBuffer",
      "createFramebuffer",
      "createProgram",
      "createRenderbuffer",
      "createShader",
      "createTexture"
    ];
			for (var ii = 0; ii < creationFunctions.length; ++ii) {
				var functionName = creationFunctions[ii];
				wrappedContext_[functionName] = function (f) {
					return function () {
						loseContextIfTime();
						if (contextLost_) {
							return null;
						}
						var obj = f.apply(ctx, arguments);
						obj.__webglDebugContextLostId__ = contextId_;
						resourceDb_.push(obj);
						return obj;
					};
				}(ctx[functionName]);
			}

			var functionsThatShouldReturnNull = [
      "getActiveAttrib",
      "getActiveUniform",
      "getBufferParameter",
      "getContextAttributes",
      "getAttachedShaders",
      "getFramebufferAttachmentParameter",
      "getParameter",
      "getProgramParameter",
      "getProgramInfoLog",
      "getRenderbufferParameter",
      "getShaderParameter",
      "getShaderInfoLog",
      "getShaderSource",
      "getTexParameter",
      "getUniform",
      "getUniformLocation",
      "getVertexAttrib"
    ];
			for (var ii = 0; ii < functionsThatShouldReturnNull.length; ++ii) {
				var functionName = functionsThatShouldReturnNull[ii];
				wrappedContext_[functionName] = function (f) {
					return function () {
						loseContextIfTime();
						if (contextLost_) {
							return null;
						}
						return f.apply(ctx, arguments);
					}
				}(wrappedContext_[functionName]);
			}

			var isFunctions = [
      "isBuffer",
      "isEnabled",
      "isFramebuffer",
      "isProgram",
      "isRenderbuffer",
      "isShader",
      "isTexture"
    ];
			for (var ii = 0; ii < isFunctions.length; ++ii) {
				var functionName = isFunctions[ii];
				wrappedContext_[functionName] = function (f) {
					return function () {
						loseContextIfTime();
						if (contextLost_) {
							return false;
						}
						return f.apply(ctx, arguments);
					}
				}(wrappedContext_[functionName]);
			}

			wrappedContext_.checkFramebufferStatus = function (f) {
				return function () {
					loseContextIfTime();
					if (contextLost_) {
						return wrappedContext_.FRAMEBUFFER_UNSUPPORTED;
					}
					return f.apply(ctx, arguments);
				};
			}(wrappedContext_.checkFramebufferStatus);

			wrappedContext_.getAttribLocation = function (f) {
				return function () {
					loseContextIfTime();
					if (contextLost_) {
						return -1;
					}
					return f.apply(ctx, arguments);
				};
			}(wrappedContext_.getAttribLocation);

			wrappedContext_.getVertexAttribOffset = function (f) {
				return function () {
					loseContextIfTime();
					if (contextLost_) {
						return 0;
					}
					return f.apply(ctx, arguments);
				};
			}(wrappedContext_.getVertexAttribOffset);

			wrappedContext_.isContextLost = function () {
				return contextLost_;
			};

			return wrappedContext_;
		}
	}

	return {
		/**
		 * Initializes this module. Safe to call more than once.
		 * @param {!WebGLRenderingContext} ctx A WebGL context. If
		 *    you have more than one context it doesn't matter which one
		 *    you pass in, it is only used to pull out constants.
		 */
		'init': init,

		/**
		 * Returns true or false if value matches any WebGL enum
		 * @param {*} value Value to check if it might be an enum.
		 * @return {boolean} True if value matches one of the WebGL defined enums
		 */
		'mightBeEnum': mightBeEnum,

		/**
		 * Gets an string version of an WebGL enum.
		 *
		 * Example:
		 *   WebGLDebugUtil.init(ctx);
		 *   var str = WebGLDebugUtil.glEnumToString(ctx.getError());
		 *
		 * @param {number} value Value to return an enum for
		 * @return {string} The string version of the enum.
		 */
		'glEnumToString': glEnumToString,

		/**
		 * Converts the argument of a WebGL function to a string.
		 * Attempts to convert enum arguments to strings.
		 *
		 * Example:
		 *   WebGLDebugUtil.init(ctx);
		 *   var str = WebGLDebugUtil.glFunctionArgToString('bindTexture', 2, 0, gl.TEXTURE_2D);
		 *
		 * would return 'TEXTURE_2D'
		 *
		 * @param {string} functionName the name of the WebGL function.
		 * @param {number} numArgs The number of arguments
		 * @param {number} argumentIndx the index of the argument.
		 * @param {*} value The value of the argument.
		 * @return {string} The value as a string.
		 */
		'glFunctionArgToString': glFunctionArgToString,

		/**
		 * Converts the arguments of a WebGL function to a string.
		 * Attempts to convert enum arguments to strings.
		 *
		 * @param {string} functionName the name of the WebGL function.
		 * @param {number} args The arguments.
		 * @return {string} The arguments as a string.
		 */
		'glFunctionArgsToString': glFunctionArgsToString,

		/**
		 * Given a WebGL context returns a wrapped context that calls
		 * gl.getError after every command and calls a function if the
		 * result is not NO_ERROR.
		 *
		 * You can supply your own function if you want. For example, if you'd like
		 * an exception thrown on any GL error you could do this
		 *
		 *    function throwOnGLError(err, funcName, args) {
		 *      throw WebGLDebugUtils.glEnumToString(err) +
		 *            " was caused by call to " + funcName;
		 *    };
		 *
		 *    ctx = WebGLDebugUtils.makeDebugContext(
		 *        canvas.getContext("webgl"), throwOnGLError);
		 *
		 * @param {!WebGLRenderingContext} ctx The webgl context to wrap.
		 * @param {!function(err, funcName, args): void} opt_onErrorFunc The function
		 *     to call when gl.getError returns an error. If not specified the default
		 *     function calls console.log with a message.
		 * @param {!function(funcName, args): void} opt_onFunc The
		 *     function to call when each webgl function is called. You
		 *     can use this to log all calls for example.
		 */
		'makeDebugContext': makeDebugContext,

		/**
		 * Given a canvas element returns a wrapped canvas element that will
		 * simulate lost context. The canvas returned adds the following functions.
		 *
		 * loseContext:
		 *   simulates a lost context event.
		 *
		 * restoreContext:
		 *   simulates the context being restored.
		 *
		 * lostContextInNCalls:
		 *   loses the context after N gl calls.
		 *
		 * getNumCalls:
		 *   tells you how many gl calls there have been so far.
		 *
		 * setRestoreTimeout:
		 *   sets the number of milliseconds until the context is restored
		 *   after it has been lost. Defaults to 0. Pass -1 to prevent
		 *   automatic restoring.
		 *
		 * @param {!Canvas} canvas The canvas element to wrap.
		 */
		'makeLostContextSimulatingCanvas': makeLostContextSimulatingCanvas,

		/**
		 * Resets a context to the initial state.
		 * @param {!WebGLRenderingContext} ctx The webgl context to
		 *     reset.
		 */
		'resetToInitialState': resetToInitialState
	};

}();


/*
 * Copyright (C) 2009 Apple Inc. All Rights Reserved.
 *
 * Redistribution and use in source and binary forms, with or without
 * modification, are permitted provided that the following conditions
 * are met:
 * 1. Redistributions of source code must retain the above copyright
 *    notice, this list of conditions and the following disclaimer.
 * 2. Redistributions in binary form must reproduce the above copyright
 *    notice, this list of conditions and the following disclaimer in the
 *    documentation and/or other materials provided with the distribution.
 *
 * THIS SOFTWARE IS PROVIDED BY APPLE INC. ``AS IS'' AND ANY
 * EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
 * IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR
 * PURPOSE ARE DISCLAIMED.  IN NO EVENT SHALL APPLE INC. OR
 * CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL,
 * EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO,
 * PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR
 * PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY
 * OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
 * (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
 * OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
 */

//
// initWebGL
//
// Initialize the Canvas element with the passed name as a WebGL object and return the
// WebGLRenderingContext.
function initWebGL(canvasName, vshader, fshader, attribs, clearColor, clearDepth) {
	var canvas = document.getElementById(canvasName);
	return gl = WebGLUtils.setupWebGL(canvas);
}

function log(msg) {
	if (window.console && window.console.log) {
		window.console.log(msg);
	}
}

// Load shaders with the passed names and create a program with them. Return this program
// in the 'program' property of the returned context.
//
// For each string in the passed attribs array, bind an attrib with that name at that index.
// Once the attribs are bound, link the program and then use it.
//
// Set the clear color to the passed array (4 values) and set the clear depth to the passed value.
// Enable depth testing and blending with a blend func of (SRC_ALPHA, ONE_MINUS_SRC_ALPHA)
//
// A console function is added to the context: console(string). This can be replaced
// by the caller. By default, it maps to the window.console() function on WebKit and to
// an empty function on other browsers.
//
function simpleSetup(gl, vshader, fshader, attribs, clearColor, clearDepth) {
	// create our shaders
	var vertexShader = loadShader(gl, vshader);
	var fragmentShader = loadShader(gl, fshader);

	// Create the program object
	var program = gl.createProgram();

	// Attach our two shaders to the program
	gl.attachShader(program, vertexShader);
	gl.attachShader(program, fragmentShader);

	// Bind attributes
	for (var i = 0; i < attribs.length; ++i)
		gl.bindAttribLocation(program, i, attribs[i]);

	// Link the program
	gl.linkProgram(program);

	// Check the link status
	var linked = gl.getProgramParameter(program, gl.LINK_STATUS);
	if (!linked && !gl.isContextLost()) {
		// something went wrong with the link
		var error = gl.getProgramInfoLog(program);
		log("Error in program linking:" + error);

		gl.deleteProgram(program);
		gl.deleteProgram(fragmentShader);
		gl.deleteProgram(vertexShader);

		return null;
	}

	gl.useProgram(program);

	gl.clearColor(clearColor[0], clearColor[1], clearColor[2], clearColor[3]);
	gl.clearDepth(clearDepth);

	gl.enable(gl.DEPTH_TEST);
	gl.enable(gl.BLEND);
	gl.blendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA);

	return program;
}

//
// loadShader
//
// 'shaderId' is the id of a <script> element containing the shader source string.
// Load this shader and return the WebGLShader object corresponding to it.
//
function loadShader(ctx, shaderId) {
	var shaderScript = document.getElementById(shaderId);
	if (!shaderScript) {
		log("*** Error: shader script '" + shaderId + "' not found");
		return null;
	}

	return loadShaderScript(ctx, shaderScript.text, shaderScript.type);
}

function loadShaderVertexScript(ctx, script) {
	return loadShaderScript(ctx, script, "x-shader/x-vertex");
}

function loadShaderFragmentScript(ctx, script) {
	return loadShaderScript(ctx, script, "x-shader/x-fragment");
}

function loadShaderScript(ctx, script, typ) {
	if (!script) {
		log("*** Error: shader script missing");
		return null;
	}

	if (typ == "x-shader/x-vertex")
		var shaderType = ctx.VERTEX_SHADER;
	else if (typ == "x-shader/x-fragment")
		var shaderType = ctx.FRAGMENT_SHADER;
	else {
		log("*** Error: shader script of undefined type '" + typ + "'");
		return null;
	}

	// Create the shader object
	var shader = ctx.createShader(shaderType);

	// Load the shader source
	ctx.shaderSource(shader, script);

	// Compile the shader
	ctx.compileShader(shader);

	// Check the compile status
	var compiled = ctx.getShaderParameter(shader, ctx.COMPILE_STATUS);
	if (!compiled && !ctx.isContextLost()) {
		// Something went wrong during compilation; get the error
		var error = ctx.getShaderInfoLog(shader);
		log("*** Error compiling shader: " + error);
		ctx.deleteShader(shader);
		return null;
	}

	return shader;
}




/* **********************************************************************************
** SAVE SOME SPACE BY COMMENTING OUT UNUSED makeAxis, makeBox, and makeSphere *******
********************************************************************************** */
/* BEGIN COMMENTING OUT makeAxis, makeBox, and makeSphere

//
// makeAxis
//
// Create a box with vertices, normals and texCoords. Create VBOs for each as well as the index array.
// Return an object with the following properties:
//
//  normalObject        WebGLBuffer object for normals
//  texCoordObject      WebGLBuffer object for texCoords
//  vertexObject        WebGLBuffer object for vertices
//  indexObject         WebGLBuffer object for indices
//  numIndices          The number of indices in the indexObject
//
function makeAxis(ctx) {
	// box
	//    v6----- v5
	//   /|      /|
	//  v1------v0|
	//  | |     | |
	//  | |v7---|-|v4
	//  |/      |/
	//  v2------v3
	//
	// vertex coords array
	var vertices = new Float32Array(
          [0, -1, 0.01,
           -1, 1, 0.01,
            1, 1, 0.01, // v0-v1-v2 front

            0.01, -1, 0,
            0.01, 1, 0.25, //  ( we reduce z from 1 to simulate the tail of an airplane)
            0.01, 1, -1, // v0-v3-v4-v5 right

            0, 0.01, -1,
            1, 0.01, 1,
           -1, 0.01, 1, // v0-v5-v6-v1 top

           -0.01, -1, 0,
           -0.01, 1, 0.25, //  ( we reduce z from 1 to simulate the tail of an airplane)
           -0.01, 1, -1, // v0-v3-v4-v5 left

            0, -0.01, -1,
            1, -0.01, 1,
           -1, -0.01, 1, // v0-v5-v6-v1 bottom

            0, -1, -0.01,
           -1, 1, -0.01,
            1, 1, -0.01] // v4-v7-v6-v5 back
	);
	// normal array
	var normals = new Float32Array(
        [0, 0, 1,
         0, 0, 1,
         0, 0, 1, // v0-v1-v2-v3 front
         1, 0, 0,
         1, 0, 0,
         1, 0, 0, // v0-v3-v4-v5 right
         0, 1, 0,
         0, 1, 0,
         0, 1, 0, // v0-v5-v6-v1 top
         -1, 0, 0,
         -1, 0, 0,
         -1, 0, 0, // v1-v6-v7-v2 left
         0, -1, 0,
         0, -1, 0,
         0, -1, 0, // v7-v4-v3-v2 bottom
         0, 0, -1,
         0, 0, -1,
         0, 0, -1] // v4-v7-v6-v5 back
	);


	// texCoord array
	var texCoords = new Float32Array(
        [1, 1, 0, 1, 0, 0, // v0-v1-v2-v3 front
           0, 1, 0, 0, 1, 0, // v0-v3-v4-v5 right
           1, 0, 1, 1, 0, 1, // v0-v5-v6-v1 top
           1, 1, 0, 1, 0, 0, // v1-v6-v7-v2 left
           0, 0, 1, 0, 1, 1, // v7-v4-v3-v2 bottom
           0, 0, 1, 0, 1, 1] // v4-v7-v6-v5 back
	);

	// index array
	var indices = new Uint8Array(
          [0, 1, 2, // front
           3, 4, 5, // right
           6, 7, 8, // top
           9, 10, 11, // left
          12, 13, 14, // bottom
          15, 16, 17] // back
	);

	// Set up the array of colors for the cube's faces
	var colors = new Uint8Array(
              [0, 1, 0, 1,
               0, 1, 0, 1,
               0, 1, 0, 1, // v0-v1-v2-v3 front : hoizontal plane

               1, 0, 0, 1,
               1, 0, 0, 1,
               1, 0, 0, 1, // v0-v3-v4-v5 right : vertical plane

               1, 1, 1, 1,
               0, 0, 1, 1,
               0, 0, 1, 1, // v0-v5-v6-v1 top : cross section (we can hide this plane by setting alpha to zero)

               1, 0, 0, 1,
               1, 0, 0, 1,
               1, 0, 0, 1, // v1-v6-v7-v2 left

               1, 1, 1, 1,
               0, 0, 1, 1,
               0, 0, 1, 1, // v7-v4-v3-v2 bottom : cross section (we can hide this plane by setting alpha to zero)

               0, 1, 0, 1,
               0, 1, 0, 1,
               0, 1, 0, 1] // v4-v7-v6-v5 back : hoizontal plane
	);


	var retval = {};

	retval.normalObject = ctx.createBuffer();
	ctx.bindBuffer(ctx.ARRAY_BUFFER, retval.normalObject);
	ctx.bufferData(ctx.ARRAY_BUFFER, normals, ctx.STATIC_DRAW);

	retval.texCoordObject = ctx.createBuffer();
	ctx.bindBuffer(ctx.ARRAY_BUFFER, retval.texCoordObject);
	ctx.bufferData(ctx.ARRAY_BUFFER, texCoords, ctx.STATIC_DRAW);

	retval.vertexObject = ctx.createBuffer();
	ctx.bindBuffer(ctx.ARRAY_BUFFER, retval.vertexObject);
	ctx.bufferData(ctx.ARRAY_BUFFER, vertices, ctx.STATIC_DRAW);

	ctx.bindBuffer(ctx.ARRAY_BUFFER, null);

	retval.indexObject = ctx.createBuffer();
	ctx.bindBuffer(ctx.ELEMENT_ARRAY_BUFFER, retval.indexObject);
	ctx.bufferData(ctx.ELEMENT_ARRAY_BUFFER, indices, ctx.STATIC_DRAW);
	ctx.bindBuffer(ctx.ELEMENT_ARRAY_BUFFER, null);

	// Set up the vertex buffer for the colors
	rtval.colorObject = gl.createBuffer();
	gl.bindBuffer(gl.ARRAY_BUFFER, rtval.colorObject);
	gl.bufferData(gl.ARRAY_BUFFER, colors, gl.STATIC_DRAW);

	retval.numIndices = indices.length;

	return retval;
}


//
// makeBox
//
// Create a box with vertices, normals and texCoords. Create VBOs for each as well as the index array.
// Return an object with the following properties:
//
//  normalObject        WebGLBuffer object for normals
//  texCoordObject      WebGLBuffer object for texCoords
//  vertexObject        WebGLBuffer object for vertices
//  indexObject         WebGLBuffer object for indices
//  numIndices          The number of indices in the indexObject
//
function makeBox(ctx) {
	// box
	//    v6----- v5
	//   /|      /|
	//  v1------v0|
	//  | |     | |
	//  | |v7---|-|v4
	//  |/      |/
	//  v2------v3
	//
	// vertex coords array
	var vertices = new Float32Array(
        [1, 1, 1, -1, 1, 1, -1, -1, 1, 1, -1, 1, // v0-v1-v2-v3 front
           1, 1, 1, 1, -1, 1, 1, -1, -1, 1, 1, -1, // v0-v3-v4-v5 right
           1, 1, 1, 1, 1, -1, -1, 1, -1, -1, 1, 1, // v0-v5-v6-v1 top
          -1, 1, 1, -1, 1, -1, -1, -1, -1, -1, -1, 1, // v1-v6-v7-v2 left
          -1, -1, -1, 1, -1, -1, 1, -1, 1, -1, -1, 1, // v7-v4-v3-v2 bottom
           1, -1, -1, -1, -1, -1, -1, 1, -1, 1, 1, -1] // v4-v7-v6-v5 back
	);

	// normal array
	var normals = new Float32Array(
        [0, 0, 1, 0, 0, 1, 0, 0, 1, 0, 0, 1, // v0-v1-v2-v3 front
           1, 0, 0, 1, 0, 0, 1, 0, 0, 1, 0, 0, // v0-v3-v4-v5 right
           0, 1, 0, 0, 1, 0, 0, 1, 0, 0, 1, 0, // v0-v5-v6-v1 top
          -1, 0, 0, -1, 0, 0, -1, 0, 0, -1, 0, 0, // v1-v6-v7-v2 left
           0, -1, 0, 0, -1, 0, 0, -1, 0, 0, -1, 0, // v7-v4-v3-v2 bottom
           0, 0, -1, 0, 0, -1, 0, 0, -1, 0, 0, -1] // v4-v7-v6-v5 back
	);

	// texCoord array
	var texCoords = new Float32Array(
        [1, 1, 0, 1, 0, 0, 1, 0, // v0-v1-v2-v3 front
           0, 1, 0, 0, 1, 0, 1, 1, // v0-v3-v4-v5 right
           1, 0, 1, 1, 0, 1, 0, 0, // v0-v5-v6-v1 top
           1, 1, 0, 1, 0, 0, 1, 0, // v1-v6-v7-v2 left
           0, 0, 1, 0, 1, 1, 0, 1, // v7-v4-v3-v2 bottom
           0, 0, 1, 0, 1, 1, 0, 1] // v4-v7-v6-v5 back
	);

	// index array
	var indices = new Uint8Array(
        [0, 1, 2, 0, 2, 3, // front
           4, 5, 6, 4, 6, 7, // right
           8, 9, 10, 8, 10, 11, // top
          12, 13, 14, 12, 14, 15, // left
          16, 17, 18, 16, 18, 19, // bottom
          20, 21, 22, 20, 22, 23] // back
	);

	var retval = {};

	retval.normalObject = ctx.createBuffer();
	ctx.bindBuffer(ctx.ARRAY_BUFFER, retval.normalObject);
	ctx.bufferData(ctx.ARRAY_BUFFER, normals, ctx.STATIC_DRAW);

	retval.texCoordObject = ctx.createBuffer();
	ctx.bindBuffer(ctx.ARRAY_BUFFER, retval.texCoordObject);
	ctx.bufferData(ctx.ARRAY_BUFFER, texCoords, ctx.STATIC_DRAW);

	retval.vertexObject = ctx.createBuffer();
	ctx.bindBuffer(ctx.ARRAY_BUFFER, retval.vertexObject);
	ctx.bufferData(ctx.ARRAY_BUFFER, vertices, ctx.STATIC_DRAW);

	ctx.bindBuffer(ctx.ARRAY_BUFFER, null);

	retval.indexObject = ctx.createBuffer();
	ctx.bindBuffer(ctx.ELEMENT_ARRAY_BUFFER, retval.indexObject);
	ctx.bufferData(ctx.ELEMENT_ARRAY_BUFFER, indices, ctx.STATIC_DRAW);
	ctx.bindBuffer(ctx.ELEMENT_ARRAY_BUFFER, null);

	retval.numIndices = indices.length;

	return retval;
}

//
// makeSphere
//
// Create a sphere with the passed number of latitude and longitude bands and the passed radius.
// Sphere has vertices, normals and texCoords. Create VBOs for each as well as the index array.
// Return an object with the following properties:
//
//  normalObject        WebGLBuffer object for normals
//  texCoordObject      WebGLBuffer object for texCoords
//  vertexObject        WebGLBuffer object for vertices
//  indexObject         WebGLBuffer object for indices
//  numIndices          The number of indices in the indexObject
//
function makeSphere(ctx, radius, lats, longs) {
	var geometryData = [];
	var normalData = [];
	var texCoordData = [];
	var indexData = [];

	for (var latNumber = 0; latNumber <= lats; ++latNumber) {
		for (var longNumber = 0; longNumber <= longs; ++longNumber) {
			var theta = latNumber * Math.PI / lats;
			var phi = longNumber * 2 * Math.PI / longs;
			var sinTheta = Math.sin(theta);
			var sinPhi = Math.sin(phi);
			var cosTheta = Math.cos(theta);
			var cosPhi = Math.cos(phi);

			var x = cosPhi * sinTheta;
			var y = cosTheta;
			var z = sinPhi * sinTheta;
			var u = 1 - (longNumber / longs);
			var v = latNumber / lats;

			normalData.push(x);
			normalData.push(y);
			normalData.push(z);
			texCoordData.push(u);
			texCoordData.push(v);
			geometryData.push(radius * x);
			geometryData.push(radius * y);
			geometryData.push(radius * z);
		}
	}

	for (var latNumber = 0; latNumber < lats; ++latNumber) {
		for (var longNumber = 0; longNumber < longs; ++longNumber) {
			var first = (latNumber * (longs + 1)) + longNumber;
			var second = first + longs + 1;
			indexData.push(first);
			indexData.push(second);
			indexData.push(first + 1);

			indexData.push(second);
			indexData.push(second + 1);
			indexData.push(first + 1);
		}
	}

	var retval = {};

	retval.normalObject = ctx.createBuffer();
	ctx.bindBuffer(ctx.ARRAY_BUFFER, retval.normalObject);
	ctx.bufferData(ctx.ARRAY_BUFFER, new Float32Array(normalData), ctx.STATIC_DRAW);

	retval.texCoordObject = ctx.createBuffer();
	ctx.bindBuffer(ctx.ARRAY_BUFFER, retval.texCoordObject);
	ctx.bufferData(ctx.ARRAY_BUFFER, new Float32Array(texCoordData), ctx.STATIC_DRAW);

	retval.vertexObject = ctx.createBuffer();
	ctx.bindBuffer(ctx.ARRAY_BUFFER, retval.vertexObject);
	ctx.bufferData(ctx.ARRAY_BUFFER, new Float32Array(geometryData), ctx.STATIC_DRAW);

	retval.numIndices = indexData.length;
	retval.indexObject = ctx.createBuffer();
	ctx.bindBuffer(ctx.ELEMENT_ARRAY_BUFFER, retval.indexObject);
	ctx.bufferData(ctx.ELEMENT_ARRAY_BUFFER, new Uint16Array(indexData), ctx.STREAM_DRAW);

	return retval;
}

END COMMENTING OUT makeAxis, makeBox, and makeSphere */

// Array of Objects curently loading
var g_loadingObjects = [];

// Clears all the Objects currently loading.
// This is used to handle context lost events.
function clearLoadingObjects() {
	for (var ii = 0; ii < g_loadingObjects.length; ++ii) {
		g_loadingObjects[ii].onreadystatechange = undefined;
	}
	g_loadingObjects = [];
}

//
// loadObj
//
// Load a .obj file from the passed URL. Return an object with a 'loaded' property set to false.
// When the object load is complete, the 'loaded' property becomes true and the following
// properties are set:
//
//  normalObject        WebGLBuffer object for normals
//  texCoordObject      WebGLBuffer object for texCoords
//  vertexObject        WebGLBuffer object for vertices
//  indexObject         WebGLBuffer object for indices
//  numIndices          The number of indices in the indexObject
//
function loadObj(ctx, url) {
	var obj = {
		loaded: false
	};
	obj.ctx = ctx;
	var req = new XMLHttpRequest();
	req.obj = obj;
	g_loadingObjects.push(req);
	req.onreadystatechange = function () {
		processLoadObj(req)
	};
	req.open("GET", url, true);
	req.send(null);
	return obj;
}

function processLoadObj(req) {
	log("req=" + req)
		// only if req shows "complete"
	if (req.readyState == 4) {
		g_loadingObjects.splice(g_loadingObjects.indexOf(req), 1);
		doLoadObj(req.obj, req.responseText);
	}
}

function doLoadObj(obj, text) {
	vertexArray = [];
	normalArray = [];
	textureArray = [];
	indexArray = [];

	var vertex = [];
	var normal = [];
	var texture = [];
	var facemap = {};
	var index = 0;

	// This is a map which associates a range of indices with a name
	// The name comes from the 'g' tag (of the form "g NAME"). Indices
	// are part of one group until another 'g' tag is seen. If any indices
	// come before a 'g' tag, it is given the group name "_unnamed"
	// 'group' is an object whose property names are the group name and
	// whose value is a 2 element array with [<first index>, <num indices>]
	var groups = {};
	var currentGroup = [-1, 0];
	groups["_unnamed"] = currentGroup;

	var lines = text.split("\n");
	for (var lineIndex in lines) {
		var line = lines[lineIndex].replace(/[ \t]+/g, " ").replace(/\s\s*$/, "");

		// ignore comments
		if (line[0] == "#")
			continue;

		var array = line.split(" ");
		if (array[0] == "g") {
			// new group
			currentGroup = [indexArray.length, 0];
			groups[array[1]] = currentGroup;
		} else if (array[0] == "v") {
			// vertex
			vertex.push(parseFloat(array[1]));
			vertex.push(parseFloat(array[2]));
			vertex.push(parseFloat(array[3]));
		} else if (array[0] == "vt") {
			// normal
			texture.push(parseFloat(array[1]));
			texture.push(parseFloat(array[2]));
		} else if (array[0] == "vn") {
			// normal
			normal.push(parseFloat(array[1]));
			normal.push(parseFloat(array[2]));
			normal.push(parseFloat(array[3]));
		} else if (array[0] == "f") {
			// face
			if (array.length != 4) {
				log("*** Error: face '" + line + "' not handled");
				continue;
			}

			for (var i = 1; i < 4; ++i) {
				if (!(array[i] in facemap)) {
					// add a new entry to the map and arrays
					var f = array[i].split("/");
					var vtx, nor, tex;

					if (f.length == 1) {
						vtx = parseInt(f[0]) - 1;
						nor = vtx;
						tex = vtx;
					} else if (f.length = 3) {
						vtx = parseInt(f[0]) - 1;
						tex = parseInt(f[1]) - 1;
						nor = parseInt(f[2]) - 1;
					} else {
						obj.ctx.console.log("*** Error: did not understand face '" + array[i] + "'");
						return null;
					}

					// do the vertices
					var x = 0;
					var y = 0;
					var z = 0;
					if (vtx * 3 + 2 < vertex.length) {
						x = vertex[vtx * 3];
						y = vertex[vtx * 3 + 1];
						z = vertex[vtx * 3 + 2];
					}
					vertexArray.push(x);
					vertexArray.push(y);
					vertexArray.push(z);

					// do the textures
					x = 0;
					y = 0;
					if (tex * 2 + 1 < texture.length) {
						x = texture[tex * 2];
						y = texture[tex * 2 + 1];
					}
					textureArray.push(x);
					textureArray.push(y);

					// do the normals
					x = 0;
					y = 0;
					z = 1;
					if (nor * 3 + 2 < normal.length) {
						x = normal[nor * 3];
						y = normal[nor * 3 + 1];
						z = normal[nor * 3 + 2];
					}
					normalArray.push(x);
					normalArray.push(y);
					normalArray.push(z);

					facemap[array[i]] = index++;
				}

				indexArray.push(facemap[array[i]]);
				currentGroup[1]++;
			}
		}
	}

	// set the VBOs
	obj.normalObject = obj.ctx.createBuffer();
	obj.ctx.bindBuffer(obj.ctx.ARRAY_BUFFER, obj.normalObject);
	obj.ctx.bufferData(obj.ctx.ARRAY_BUFFER, new Float32Array(normalArray), obj.ctx.STATIC_DRAW);

	obj.texCoordObject = obj.ctx.createBuffer();
	obj.ctx.bindBuffer(obj.ctx.ARRAY_BUFFER, obj.texCoordObject);
	obj.ctx.bufferData(obj.ctx.ARRAY_BUFFER, new Float32Array(textureArray), obj.ctx.STATIC_DRAW);

	obj.vertexObject = obj.ctx.createBuffer();
	obj.ctx.bindBuffer(obj.ctx.ARRAY_BUFFER, obj.vertexObject);
	obj.ctx.bufferData(obj.ctx.ARRAY_BUFFER, new Float32Array(vertexArray), obj.ctx.STATIC_DRAW);

	obj.numIndices = indexArray.length;
	obj.indexObject = obj.ctx.createBuffer();
	obj.ctx.bindBuffer(obj.ctx.ELEMENT_ARRAY_BUFFER, obj.indexObject);
	obj.ctx.bufferData(obj.ctx.ELEMENT_ARRAY_BUFFER, new Uint16Array(indexArray), obj.ctx.STREAM_DRAW);

	obj.groups = groups;

	obj.loaded = true;
}

// Array of images curently loading
var g_loadingImages = [];

// Clears all the images currently loading.
// This is used to handle context lost events.
function clearLoadingImages() {
	for (var ii = 0; ii < g_loadingImages.length; ++ii) {
		g_loadingImages[ii].onload = undefined;
	}
	g_loadingImages = [];
}

//
// loadImageTexture
//
// Load the image at the passed url, place it in a new WebGLTexture object and return the WebGLTexture.
//
function loadImageTexture(ctx, url) {
	var texture = ctx.createTexture();
	ctx.bindTexture(ctx.TEXTURE_2D, texture);
	ctx.texImage2D(ctx.TEXTURE_2D, 0, ctx.RGBA, 1, 1, 0, ctx.RGBA, ctx.UNSIGNED_BYTE, null);
	var image = new Image();
	g_loadingImages.push(image);
	image.onload = function () {
		doLoadImageTexture(ctx, image, texture)
	}
	image.src = url;
	return texture;
}

function doLoadImageTexture(ctx, image, texture) {
	g_loadingImages.splice(g_loadingImages.indexOf(image), 1);
	ctx.bindTexture(ctx.TEXTURE_2D, texture);
	ctx.texImage2D(
		ctx.TEXTURE_2D, 0, ctx.RGBA, ctx.RGBA, ctx.UNSIGNED_BYTE, image);
	ctx.texParameteri(ctx.TEXTURE_2D, ctx.TEXTURE_MAG_FILTER, ctx.LINEAR);
	ctx.texParameteri(ctx.TEXTURE_2D, ctx.TEXTURE_MIN_FILTER, ctx.LINEAR);
	ctx.texParameteri(ctx.TEXTURE_2D, ctx.TEXTURE_WRAP_S, ctx.CLAMP_TO_EDGE);
	ctx.texParameteri(ctx.TEXTURE_2D, ctx.TEXTURE_WRAP_T, ctx.CLAMP_TO_EDGE);
	//ctx.generateMipmap(ctx.TEXTURE_2D)
	ctx.bindTexture(ctx.TEXTURE_2D, null);
}

//
// Framerate object
//
// This object keeps track of framerate and displays it as the innerHTML text of the
// HTML element with the passed id. Once created you call snapshot at the end
// of every rendering cycle. Every 500ms the framerate is updated in the HTML element.
//
Framerate = function (id) {
	this.numFramerates = 10;
	this.framerateUpdateInterval = 500;
	this.id = id;

	this.renderTime = -1;
	this.framerates = [];
	self = this;
	var fr = function () {
		self.updateFramerate()
	}
	setInterval(fr, this.framerateUpdateInterval);
}

Framerate.prototype.updateFramerate = function () {
	var tot = 0;
	for (var i = 0; i < this.framerates.length; ++i)
		tot += this.framerates[i];

	var framerate = tot / this.framerates.length;
	framerate = Math.round(framerate);
	document.getElementById(this.id).innerHTML = "Framerate:" + framerate + "fps";
}

Framerate.prototype.snapshot = function () {
	if (this.renderTime < 0)
		this.renderTime = new Date().getTime();
	else {
		var newTime = new Date().getTime();
		var t = newTime - this.renderTime;
		if (t == 0)
			return;
		var framerate = 1000 / t;
		this.framerates.push(framerate);
		while (this.framerates.length > this.numFramerates)
			this.framerates.shift();
		this.renderTime = newTime;
	}
}









/*
 * Copyright (C) 2009 Apple Inc. All Rights Reserved.
 *
 * Redistribution and use in source and binary forms, with or without
 * modification, are permitted provided that the following conditions
 * are met:
 * 1. Redistributions of source code must retain the above copyright
 *    notice, this list of conditions and the following disclaimer.
 * 2. Redistributions in binary form must reproduce the above copyright
 *    notice, this list of conditions and the following disclaimer in the
 *    documentation and/or other materials provided with the distribution.
 *
 * THIS SOFTWARE IS PROVIDED BY APPLE INC. ``AS IS'' AND ANY
 * EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
 * IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR
 * PURPOSE ARE DISCLAIMED.  IN NO EVENT SHALL APPLE INC. OR
 * CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL,
 * EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO,
 * PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR
 * PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY
 * OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
 * (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
 * OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
 */

// J3DI (Jedi) - A support library for WebGL.

/*
    J3DI Math Classes. Currently includes:

        J3DIMatrix4 - A 4x4 Matrix
*/

/*
    J3DIMatrix4 class

    This class implements a 4x4 matrix. It has functions which duplicate the
    functionality of the OpenGL matrix stack and glut functions. On browsers
    that support it, CSSMatrix is used to accelerate operations.

    IDL:

    [
        Constructor(in J3DIMatrix4 matrix),                 // copy passed matrix into new J3DIMatrix4
        Constructor(in sequence<float> array)               // create new J3DIMatrix4 with 16 floats (column major)
        Constructor()                                       // create new J3DIMatrix4 with identity matrix
    ]
    interface J3DIMatrix4 {
        void load(in J3DIMatrix4 matrix);                   // copy the values from the passed matrix
        void load(in sequence<float> array);                // copy 16 floats into the matrix
        sequence<float> getAsArray();                       // return the matrix as an array of 16 floats
        Float32Array getAsFloat32Array();                   // return the matrix as a Float32Array with 16 values
        void setUniform(in WebGLRenderingContext ctx,       // Send the matrix to the passed uniform location in the passed context
                        in WebGLUniformLocation loc,
                        in boolean transpose);
        void makeIdentity();                                // replace the matrix with identity
        void transpose();                                   // replace the matrix with its transpose
        void invert();                                      // replace the matrix with its inverse

        void translate(in float x, in float y, in float z); // multiply the matrix by passed translation values on the right
        void translate(in J3DIVector3 v);                   // multiply the matrix by passed translation values on the right
        void scale(in float x, in float y, in float z);     // multiply the matrix by passed scale values on the right
        void scale(in J3DIVector3 v);                       // multiply the matrix by passed scale values on the right
        void rotate(in float angle,                         // multiply the matrix by passed rotation values on the right
                    in float x, in float y, in float z);    // (angle is in degrees)
        void rotate(in float angle, in J3DIVector3 v);      // multiply the matrix by passed rotation values on the right
                                                            // (angle is in degrees)
        void multiply(in CanvasMatrix matrix);              // multiply the matrix by the passed matrix on the right
        void divide(in float divisor);                      // divide the matrix by the passed divisor
        void ortho(in float left, in float right,           // multiply the matrix by the passed ortho values on the right
                   in float bottom, in float top,
                   in float near, in float far);
        void frustum(in float left, in float right,         // multiply the matrix by the passed frustum values on the right
                     in float bottom, in float top,
                     in float near, in float far);
        void perspective(in float fovy, in float aspect,    // multiply the matrix by the passed perspective values on the right
                         in float zNear, in float zFar);
        void lookat(in J3DIVector3 eye,                     // multiply the matrix by the passed lookat values on the right
                in J3DIVector3 center,  in J3DIVector3 up);
        bool decompose(in J3DIVector3 translate,            // decompose the matrix into the passed vectors
                       in J3DIVector3 rotate,
                       in J3DIVector3 scale,
                       in J3DIVector3 skew,
                       in sequence<float> perspective);
    }

    [
        Constructor(in J3DIVector3 vector),                 // copy passed vector into new J3DIVector3
        Constructor(in sequence<float> array)               // create new J3DIVector3 with 3 floats from array
        Constructor(in float x, in float y, in float z)     // create new J3DIVector3 with 3 floats
        Constructor()                                       // create new J3DIVector3 with (0,0,0)
    ]
    interface J3DIVector3 {
        void load(in J3DIVector3 vector);                   // copy the values from the passed vector
        void load(in sequence<float> array);                // copy 3 floats into the vector from array
        void load(in float x, in float y, in float z);      // copy 3 floats into the vector
        sequence<float> getAsArray();                       // return the vector as an array of 3 floats
        Float32Array getAsFloat32Array();                   // return the vector as a Float32Array with 3 values
        void multVecMatrix(in J3DIMatrix4 matrix);          // transform the vector with the passed matrix containing a homogenous coordinate transform
        float vectorLength();                               // return the length of the vector
        float dot(in J3DIVector3 v);                        // return the dot product vector . v
        void cross(in J3DIVector3 v);                       // replace the vector with cross product vector x v
        void divide(in float divisor);                      // divide the vector by the passed divisor
    }
*/


J3DIHasCSSMatrix = false;
J3DIHasCSSMatrixCopy = false;
/*
if ("WebKitCSSMatrix" in window && ("media" in window && window.media.matchMedium("(-webkit-transform-3d)")) ||
                                   ("styleMedia" in window && window.styleMedia.matchMedium("(-webkit-transform-3d)"))) {
    J3DIHasCSSMatrix = true;
    if ("copy" in WebKitCSSMatrix.prototype)
        J3DIHasCSSMatrixCopy = true;
}
*/

//  console.log("J3DIHasCSSMatrix="+J3DIHasCSSMatrix);
//  console.log("J3DIHasCSSMatrixCopy="+J3DIHasCSSMatrixCopy);

//
// J3DIMatrix4
//
J3DIMatrix4 = function (m) {
	if (J3DIHasCSSMatrix)
		this.$matrix = new WebKitCSSMatrix;
	else
		this.$matrix = new Object;

	if (typeof m == 'object') {
		if ("length" in m && m.length >= 16) {
			this.load(m);
			return;
		} else if (m instanceof J3DIMatrix4) {
			this.load(m);
			return;
		}
	}
	this.makeIdentity();
}

J3DIMatrix4.prototype.load = function () {
	if (arguments.length == 1 && typeof arguments[0] == 'object') {
		var matrix;

		if (arguments[0] instanceof J3DIMatrix4) {
			matrix = arguments[0].$matrix;

			this.$matrix.m11 = matrix.m11;
			this.$matrix.m12 = matrix.m12;
			this.$matrix.m13 = matrix.m13;
			this.$matrix.m14 = matrix.m14;

			this.$matrix.m21 = matrix.m21;
			this.$matrix.m22 = matrix.m22;
			this.$matrix.m23 = matrix.m23;
			this.$matrix.m24 = matrix.m24;

			this.$matrix.m31 = matrix.m31;
			this.$matrix.m32 = matrix.m32;
			this.$matrix.m33 = matrix.m33;
			this.$matrix.m34 = matrix.m34;

			this.$matrix.m41 = matrix.m41;
			this.$matrix.m42 = matrix.m42;
			this.$matrix.m43 = matrix.m43;
			this.$matrix.m44 = matrix.m44;
			return;
		} else
			matrix = arguments[0];

		if ("length" in matrix && matrix.length >= 16) {
			this.$matrix.m11 = matrix[0];
			this.$matrix.m12 = matrix[1];
			this.$matrix.m13 = matrix[2];
			this.$matrix.m14 = matrix[3];

			this.$matrix.m21 = matrix[4];
			this.$matrix.m22 = matrix[5];
			this.$matrix.m23 = matrix[6];
			this.$matrix.m24 = matrix[7];

			this.$matrix.m31 = matrix[8];
			this.$matrix.m32 = matrix[9];
			this.$matrix.m33 = matrix[10];
			this.$matrix.m34 = matrix[11];

			this.$matrix.m41 = matrix[12];
			this.$matrix.m42 = matrix[13];
			this.$matrix.m43 = matrix[14];
			this.$matrix.m44 = matrix[15];
			return;
		}
	}

	this.makeIdentity();
}

J3DIMatrix4.prototype.getAsArray = function () {
	return [
        this.$matrix.m11, this.$matrix.m12, this.$matrix.m13, this.$matrix.m14,
        this.$matrix.m21, this.$matrix.m22, this.$matrix.m23, this.$matrix.m24,
        this.$matrix.m31, this.$matrix.m32, this.$matrix.m33, this.$matrix.m34,
        this.$matrix.m41, this.$matrix.m42, this.$matrix.m43, this.$matrix.m44
    ];
}

J3DIMatrix4.prototype.getAsFloat32Array = function () {
	if (J3DIHasCSSMatrixCopy) {
		var array = new Float32Array(16);
		this.$matrix.copy(array);
		return array;
	}
	return new Float32Array(this.getAsArray());
}

J3DIMatrix4.prototype.setUniform = function (ctx, loc, transpose) {
	if (J3DIMatrix4.setUniformArray == undefined) {
		J3DIMatrix4.setUniformWebGLArray = new Float32Array(16);
		J3DIMatrix4.setUniformArray = new Array(16);
	}

	if (J3DIHasCSSMatrixCopy)
		this.$matrix.copy(J3DIMatrix4.setUniformWebGLArray);
	else {
		J3DIMatrix4.setUniformArray[0] = this.$matrix.m11;
		J3DIMatrix4.setUniformArray[1] = this.$matrix.m12;
		J3DIMatrix4.setUniformArray[2] = this.$matrix.m13;
		J3DIMatrix4.setUniformArray[3] = this.$matrix.m14;
		J3DIMatrix4.setUniformArray[4] = this.$matrix.m21;
		J3DIMatrix4.setUniformArray[5] = this.$matrix.m22;
		J3DIMatrix4.setUniformArray[6] = this.$matrix.m23;
		J3DIMatrix4.setUniformArray[7] = this.$matrix.m24;
		J3DIMatrix4.setUniformArray[8] = this.$matrix.m31;
		J3DIMatrix4.setUniformArray[9] = this.$matrix.m32;
		J3DIMatrix4.setUniformArray[10] = this.$matrix.m33;
		J3DIMatrix4.setUniformArray[11] = this.$matrix.m34;
		J3DIMatrix4.setUniformArray[12] = this.$matrix.m41;
		J3DIMatrix4.setUniformArray[13] = this.$matrix.m42;
		J3DIMatrix4.setUniformArray[14] = this.$matrix.m43;
		J3DIMatrix4.setUniformArray[15] = this.$matrix.m44;

		J3DIMatrix4.setUniformWebGLArray.set(J3DIMatrix4.setUniformArray);
	}

	ctx.uniformMatrix4fv(loc, transpose, J3DIMatrix4.setUniformWebGLArray);
}

J3DIMatrix4.prototype.makeIdentity = function () {
	this.$matrix.m11 = 1;
	this.$matrix.m12 = 0;
	this.$matrix.m13 = 0;
	this.$matrix.m14 = 0;

	this.$matrix.m21 = 0;
	this.$matrix.m22 = 1;
	this.$matrix.m23 = 0;
	this.$matrix.m24 = 0;

	this.$matrix.m31 = 0;
	this.$matrix.m32 = 0;
	this.$matrix.m33 = 1;
	this.$matrix.m34 = 0;

	this.$matrix.m41 = 0;
	this.$matrix.m42 = 0;
	this.$matrix.m43 = 0;
	this.$matrix.m44 = 1;
}

J3DIMatrix4.prototype.transpose = function () {
	var tmp = this.$matrix.m12;
	this.$matrix.m12 = this.$matrix.m21;
	this.$matrix.m21 = tmp;

	tmp = this.$matrix.m13;
	this.$matrix.m13 = this.$matrix.m31;
	this.$matrix.m31 = tmp;

	tmp = this.$matrix.m14;
	this.$matrix.m14 = this.$matrix.m41;
	this.$matrix.m41 = tmp;

	tmp = this.$matrix.m23;
	this.$matrix.m23 = this.$matrix.m32;
	this.$matrix.m32 = tmp;

	tmp = this.$matrix.m24;
	this.$matrix.m24 = this.$matrix.m42;
	this.$matrix.m42 = tmp;

	tmp = this.$matrix.m34;
	this.$matrix.m34 = this.$matrix.m43;
	this.$matrix.m43 = tmp;
}

J3DIMatrix4.prototype.invert = function () {
	if (J3DIHasCSSMatrix) {
		this.$matrix = this.$matrix.inverse();
		return;
	}

	// Calculate the 4x4 determinant
	// If the determinant is zero,
	// then the inverse matrix is not unique.
	var det = this._determinant4x4();

	if (Math.abs(det) < 1e-8)
		return null;

	this._makeAdjoint();

	// Scale the adjoint matrix to get the inverse
	this.$matrix.m11 /= det;
	this.$matrix.m12 /= det;
	this.$matrix.m13 /= det;
	this.$matrix.m14 /= det;

	this.$matrix.m21 /= det;
	this.$matrix.m22 /= det;
	this.$matrix.m23 /= det;
	this.$matrix.m24 /= det;

	this.$matrix.m31 /= det;
	this.$matrix.m32 /= det;
	this.$matrix.m33 /= det;
	this.$matrix.m34 /= det;

	this.$matrix.m41 /= det;
	this.$matrix.m42 /= det;
	this.$matrix.m43 /= det;
	this.$matrix.m44 /= det;
}

J3DIMatrix4.prototype.translate = function (x, y, z) {
	if (typeof x == 'object' && "length" in x) {
		var t = x;
		x = t[0];
		y = t[1];
		z = t[2];
	} else {
		if (x == undefined)
			x = 0;
		if (y == undefined)
			y = 0;
		if (z == undefined)
			z = 0;
	}

	if (J3DIHasCSSMatrix) {
		this.$matrix = this.$matrix.translate(x, y, z);
		return;
	}

	var matrix = new J3DIMatrix4();
	matrix.$matrix.m41 = x;
	matrix.$matrix.m42 = y;
	matrix.$matrix.m43 = z;

	this.multiply(matrix);
}

J3DIMatrix4.prototype.scale = function (x, y, z) {
	if (typeof x == 'object' && "length" in x) {
		var t = x;
		x = t[0];
		y = t[1];
		z = t[2];
	} else {
		if (x == undefined)
			x = 1;
		if (z == undefined) {
			if (y == undefined) {
				y = x;
				z = x;
			} else
				z = 1;
		} else if (y == undefined)
			y = x;
	}

	if (J3DIHasCSSMatrix) {
		this.$matrix = this.$matrix.scale(x, y, z);
		return;
	}

	var matrix = new J3DIMatrix4();
	matrix.$matrix.m11 = x;
	matrix.$matrix.m22 = y;
	matrix.$matrix.m33 = z;

	this.multiply(matrix);
}

J3DIMatrix4.prototype.rotate = function (angle, x, y, z) {
	// Forms are (angle, x,y,z), (angle,vector), (angleX, angleY, angleZ), (angle)
	if (typeof x == 'object' && "length" in x) {
		var t = x;
		x = t[0];
		y = t[1];
		z = t[2];
	} else {
		if (arguments.length == 1) {
			x = 0;
			y = 0;
			z = 1;
		} else if (arguments.length == 3) {
			this.rotate(angle, 1, 0, 0); // about X axis
			this.rotate(x, 0, 1, 0); // about Y axis
			this.rotate(y, 0, 0, 1); // about Z axis
			return;
		}
	}

	if (J3DIHasCSSMatrix) {
		this.$matrix = this.$matrix.rotateAxisAngle(x, y, z, angle);
		return;
	}

	// angles are in degrees. Switch to radians
	angle = angle / 180 * Math.PI;

	angle /= 2;
	var sinA = Math.sin(angle);
	var cosA = Math.cos(angle);
	var sinA2 = sinA * sinA;

	// normalize
	var len = Math.sqrt(x * x + y * y + z * z);
	if (len == 0) {
		// bad vector, just use something reasonable
		x = 0;
		y = 0;
		z = 1;
	} else if (len != 1) {
		x /= len;
		y /= len;
		z /= len;
	}

	var mat = new J3DIMatrix4();

	// optimize case where axis is along major axis
	if (x == 1 && y == 0 && z == 0) {
		mat.$matrix.m11 = 1;
		mat.$matrix.m12 = 0;
		mat.$matrix.m13 = 0;
		mat.$matrix.m21 = 0;
		mat.$matrix.m22 = 1 - 2 * sinA2;
		mat.$matrix.m23 = 2 * sinA * cosA;
		mat.$matrix.m31 = 0;
		mat.$matrix.m32 = -2 * sinA * cosA;
		mat.$matrix.m33 = 1 - 2 * sinA2;
		mat.$matrix.m14 = mat.$matrix.m24 = mat.$matrix.m34 = 0;
		mat.$matrix.m41 = mat.$matrix.m42 = mat.$matrix.m43 = 0;
		mat.$matrix.m44 = 1;
	} else if (x == 0 && y == 1 && z == 0) {
		mat.$matrix.m11 = 1 - 2 * sinA2;
		mat.$matrix.m12 = 0;
		mat.$matrix.m13 = -2 * sinA * cosA;
		mat.$matrix.m21 = 0;
		mat.$matrix.m22 = 1;
		mat.$matrix.m23 = 0;
		mat.$matrix.m31 = 2 * sinA * cosA;
		mat.$matrix.m32 = 0;
		mat.$matrix.m33 = 1 - 2 * sinA2;
		mat.$matrix.m14 = mat.$matrix.m24 = mat.$matrix.m34 = 0;
		mat.$matrix.m41 = mat.$matrix.m42 = mat.$matrix.m43 = 0;
		mat.$matrix.m44 = 1;
	} else if (x == 0 && y == 0 && z == 1) {
		mat.$matrix.m11 = 1 - 2 * sinA2;
		mat.$matrix.m12 = 2 * sinA * cosA;
		mat.$matrix.m13 = 0;
		mat.$matrix.m21 = -2 * sinA * cosA;
		mat.$matrix.m22 = 1 - 2 * sinA2;
		mat.$matrix.m23 = 0;
		mat.$matrix.m31 = 0;
		mat.$matrix.m32 = 0;
		mat.$matrix.m33 = 1;
		mat.$matrix.m14 = mat.$matrix.m24 = mat.$matrix.m34 = 0;
		mat.$matrix.m41 = mat.$matrix.m42 = mat.$matrix.m43 = 0;
		mat.$matrix.m44 = 1;
	} else {
		var x2 = x * x;
		var y2 = y * y;
		var z2 = z * z;

		mat.$matrix.m11 = 1 - 2 * (y2 + z2) * sinA2;
		mat.$matrix.m12 = 2 * (x * y * sinA2 + z * sinA * cosA);
		mat.$matrix.m13 = 2 * (x * z * sinA2 - y * sinA * cosA);
		mat.$matrix.m21 = 2 * (y * x * sinA2 - z * sinA * cosA);
		mat.$matrix.m22 = 1 - 2 * (z2 + x2) * sinA2;
		mat.$matrix.m23 = 2 * (y * z * sinA2 + x * sinA * cosA);
		mat.$matrix.m31 = 2 * (z * x * sinA2 + y * sinA * cosA);
		mat.$matrix.m32 = 2 * (z * y * sinA2 - x * sinA * cosA);
		mat.$matrix.m33 = 1 - 2 * (x2 + y2) * sinA2;
		mat.$matrix.m14 = mat.$matrix.m24 = mat.$matrix.m34 = 0;
		mat.$matrix.m41 = mat.$matrix.m42 = mat.$matrix.m43 = 0;
		mat.$matrix.m44 = 1;
	}
	this.multiply(mat);
}

J3DIMatrix4.prototype.multiply = function (mat) {
	if (J3DIHasCSSMatrix) {
		this.$matrix = this.$matrix.multiply(mat.$matrix);
		return;
	}

	// Note that m12 is the value in the first column and second row, etc.

	var m11 = (mat.$matrix.m11 * this.$matrix.m11 + mat.$matrix.m12 * this.$matrix.m21 + mat.$matrix.m13 * this.$matrix.m31 + mat.$matrix.m14 * this.$matrix.m41);
	var m12 = (mat.$matrix.m11 * this.$matrix.m12 + mat.$matrix.m12 * this.$matrix.m22 + mat.$matrix.m13 * this.$matrix.m32 + mat.$matrix.m14 * this.$matrix.m42);
	var m13 = (mat.$matrix.m11 * this.$matrix.m13 + mat.$matrix.m12 * this.$matrix.m23 + mat.$matrix.m13 * this.$matrix.m33 + mat.$matrix.m14 * this.$matrix.m43);
	var m14 = (mat.$matrix.m11 * this.$matrix.m14 + mat.$matrix.m12 * this.$matrix.m24 + mat.$matrix.m13 * this.$matrix.m34 + mat.$matrix.m14 * this.$matrix.m44);

	var m21 = (mat.$matrix.m21 * this.$matrix.m11 + mat.$matrix.m22 * this.$matrix.m21 + mat.$matrix.m23 * this.$matrix.m31 + mat.$matrix.m24 * this.$matrix.m41);
	var m22 = (mat.$matrix.m21 * this.$matrix.m12 + mat.$matrix.m22 * this.$matrix.m22 + mat.$matrix.m23 * this.$matrix.m32 + mat.$matrix.m24 * this.$matrix.m42);
	var m23 = (mat.$matrix.m21 * this.$matrix.m13 + mat.$matrix.m22 * this.$matrix.m23 + mat.$matrix.m23 * this.$matrix.m33 + mat.$matrix.m24 * this.$matrix.m43);
	var m24 = (mat.$matrix.m21 * this.$matrix.m14 + mat.$matrix.m22 * this.$matrix.m24 + mat.$matrix.m23 * this.$matrix.m34 + mat.$matrix.m24 * this.$matrix.m44);

	var m31 = (mat.$matrix.m31 * this.$matrix.m11 + mat.$matrix.m32 * this.$matrix.m21 + mat.$matrix.m33 * this.$matrix.m31 + mat.$matrix.m34 * this.$matrix.m41);
	var m32 = (mat.$matrix.m31 * this.$matrix.m12 + mat.$matrix.m32 * this.$matrix.m22 + mat.$matrix.m33 * this.$matrix.m32 + mat.$matrix.m34 * this.$matrix.m42);
	var m33 = (mat.$matrix.m31 * this.$matrix.m13 + mat.$matrix.m32 * this.$matrix.m23 + mat.$matrix.m33 * this.$matrix.m33 + mat.$matrix.m34 * this.$matrix.m43);
	var m34 = (mat.$matrix.m31 * this.$matrix.m14 + mat.$matrix.m32 * this.$matrix.m24 + mat.$matrix.m33 * this.$matrix.m34 + mat.$matrix.m34 * this.$matrix.m44);

	var m41 = (mat.$matrix.m41 * this.$matrix.m11 + mat.$matrix.m42 * this.$matrix.m21 + mat.$matrix.m43 * this.$matrix.m31 + mat.$matrix.m44 * this.$matrix.m41);
	var m42 = (mat.$matrix.m41 * this.$matrix.m12 + mat.$matrix.m42 * this.$matrix.m22 + mat.$matrix.m43 * this.$matrix.m32 + mat.$matrix.m44 * this.$matrix.m42);
	var m43 = (mat.$matrix.m41 * this.$matrix.m13 + mat.$matrix.m42 * this.$matrix.m23 + mat.$matrix.m43 * this.$matrix.m33 + mat.$matrix.m44 * this.$matrix.m43);
	var m44 = (mat.$matrix.m41 * this.$matrix.m14 + mat.$matrix.m42 * this.$matrix.m24 + mat.$matrix.m43 * this.$matrix.m34 + mat.$matrix.m44 * this.$matrix.m44);

	this.$matrix.m11 = m11;
	this.$matrix.m12 = m12;
	this.$matrix.m13 = m13;
	this.$matrix.m14 = m14;

	this.$matrix.m21 = m21;
	this.$matrix.m22 = m22;
	this.$matrix.m23 = m23;
	this.$matrix.m24 = m24;

	this.$matrix.m31 = m31;
	this.$matrix.m32 = m32;
	this.$matrix.m33 = m33;
	this.$matrix.m34 = m34;

	this.$matrix.m41 = m41;
	this.$matrix.m42 = m42;
	this.$matrix.m43 = m43;
	this.$matrix.m44 = m44;
}

J3DIMatrix4.prototype.divide = function (divisor) {
	this.$matrix.m11 /= divisor;
	this.$matrix.m12 /= divisor;
	this.$matrix.m13 /= divisor;
	this.$matrix.m14 /= divisor;

	this.$matrix.m21 /= divisor;
	this.$matrix.m22 /= divisor;
	this.$matrix.m23 /= divisor;
	this.$matrix.m24 /= divisor;

	this.$matrix.m31 /= divisor;
	this.$matrix.m32 /= divisor;
	this.$matrix.m33 /= divisor;
	this.$matrix.m34 /= divisor;

	this.$matrix.m41 /= divisor;
	this.$matrix.m42 /= divisor;
	this.$matrix.m43 /= divisor;
	this.$matrix.m44 /= divisor;

}

J3DIMatrix4.prototype.ortho = function (left, right, bottom, top, near, far) {
	var tx = (left + right) / (left - right);
	var ty = (top + bottom) / (top - bottom);
	var tz = (far + near) / (far - near);

	var matrix = new J3DIMatrix4();
	matrix.$matrix.m11 = 2 / (left - right);
	matrix.$matrix.m12 = 0;
	matrix.$matrix.m13 = 0;
	matrix.$matrix.m14 = 0;
	matrix.$matrix.m21 = 0;
	matrix.$matrix.m22 = 2 / (top - bottom);
	matrix.$matrix.m23 = 0;
	matrix.$matrix.m24 = 0;
	matrix.$matrix.m31 = 0;
	matrix.$matrix.m32 = 0;
	matrix.$matrix.m33 = -2 / (far - near);
	matrix.$matrix.m34 = 0;
	matrix.$matrix.m41 = tx;
	matrix.$matrix.m42 = ty;
	matrix.$matrix.m43 = tz;
	matrix.$matrix.m44 = 1;

	this.multiply(matrix);
}

J3DIMatrix4.prototype.frustum = function (left, right, bottom, top, near, far) {
	var matrix = new J3DIMatrix4();
	var A = (right + left) / (right - left);
	var B = (top + bottom) / (top - bottom);
	var C = -(far + near) / (far - near);
	var D = -(2 * far * near) / (far - near);

	matrix.$matrix.m11 = (2 * near) / (right - left);
	matrix.$matrix.m12 = 0;
	matrix.$matrix.m13 = 0;
	matrix.$matrix.m14 = 0;

	matrix.$matrix.m21 = 0;
	matrix.$matrix.m22 = 2 * near / (top - bottom);
	matrix.$matrix.m23 = 0;
	matrix.$matrix.m24 = 0;

	matrix.$matrix.m31 = A;
	matrix.$matrix.m32 = B;
	matrix.$matrix.m33 = C;
	matrix.$matrix.m34 = -1;

	matrix.$matrix.m41 = 0;
	matrix.$matrix.m42 = 0;
	matrix.$matrix.m43 = D;
	matrix.$matrix.m44 = 0;

	this.multiply(matrix);
}

J3DIMatrix4.prototype.perspective = function (fovy, aspect, zNear, zFar) {
	var top = Math.tan(fovy * Math.PI / 360) * zNear;
	var bottom = -top;
	var left = aspect * bottom;
	var right = aspect * top;
	this.frustum(left, right, bottom, top, zNear, zFar);
}

J3DIMatrix4.prototype.lookat = function (eyex, eyey, eyez, centerx, centery, centerz, upx, upy, upz) {
	if (typeof eyez == 'object' && "length" in eyez) {
		var t = eyez;
		upx = t[0];
		upy = t[1];
		upz = t[2];

		t = eyey;
		centerx = t[0];
		centery = t[1];
		centerz = t[2];

		t = eyex;
		eyex = t[0];
		eyey = t[1];
		eyez = t[2];
	}

	var matrix = new J3DIMatrix4();

	// Make rotation matrix

	// Z vector
	var zx = eyex - centerx;
	var zy = eyey - centery;
	var zz = eyez - centerz;
	var mag = Math.sqrt(zx * zx + zy * zy + zz * zz);
	if (mag) {
		zx /= mag;
		zy /= mag;
		zz /= mag;
	}

	// Y vector
	var yx = upx;
	var yy = upy;
	var yz = upz;

	// X vector = Y cross Z
	xx = yy * zz - yz * zy;
	xy = -yx * zz + yz * zx;
	xz = yx * zy - yy * zx;

	// Recompute Y = Z cross X
	yx = zy * xz - zz * xy;
	yy = -zx * xz + zz * xx;
	yx = zx * xy - zy * xx;

	// cross product gives area of parallelogram, which is < 1.0 for
	// non-perpendicular unit-length vectors; so normalize x, y here

	mag = Math.sqrt(xx * xx + xy * xy + xz * xz);
	if (mag) {
		xx /= mag;
		xy /= mag;
		xz /= mag;
	}

	mag = Math.sqrt(yx * yx + yy * yy + yz * yz);
	if (mag) {
		yx /= mag;
		yy /= mag;
		yz /= mag;
	}

	matrix.$matrix.m11 = xx;
	matrix.$matrix.m12 = xy;
	matrix.$matrix.m13 = xz;
	matrix.$matrix.m14 = 0;

	matrix.$matrix.m21 = yx;
	matrix.$matrix.m22 = yy;
	matrix.$matrix.m23 = yz;
	matrix.$matrix.m24 = 0;

	matrix.$matrix.m31 = zx;
	matrix.$matrix.m32 = zy;
	matrix.$matrix.m33 = zz;
	matrix.$matrix.m34 = 0;

	matrix.$matrix.m41 = 0;
	matrix.$matrix.m42 = 0;
	matrix.$matrix.m43 = 0;
	matrix.$matrix.m44 = 1;
	matrix.translate(-eyex, -eyey, -eyez);

	this.multiply(matrix);
}

// Decompose the matrix to the passed vectors. Returns true on success, false
// otherwise. All params are Array objects.
// Based on James Arvo: Graphics Gems II section VII. 1 Decomposing a matrix
// into simple transformations. Source code here:
// http://tog.acm.org/resources/GraphicsGems/gemsii/unmatrix.c
// The rotation decomposition code in the book is incorrect, official errata
// is here: http://tog.acm.org/resources/GraphicsGems/Errata.GraphicsGemsII
//
// This code has completely re-derived rotation decomposition since the book
// has different conventions for the handedness of rotations, and the
// explanation in the errata is not very thorough either.
//
// Rotation matrix Rx * Ry * Rz = rotate(A, B, C)
//
//   [ 1  0       0       ]   [ cos(B)   0  sin(B) ]   [ cos(C)  -sin(C)  0 ]
// = | 0  cos(A)  -sin(A) | * | 0        1  0      | * | sin(C)  cos(C)   0 |
//   [ 0  sin(A)  cos(A)  ]   [ -sin(B)  0  cos(B) ]   [ 0       0        1 ]
//
//   [ cos(B)*cos(C)                          -cos(B)*sin(C)                         sin(B)         ]
// = | sin(A)*sin(B)*cos(C) + cos(A)*sin(C)   -sin(A)*sin(B)*sin(C) + cos(A)*cos(C)  -sin(A)*cos(B) |
//   [ -cos(A)*sin(B)*cos(C) + sin(A)*sin(C)  cos(A)*sin(B)*sin(C) + sin(A)*cos(C)   cos(A)*cos(B)  ]
//
// From here, we easily get B = asin(m31) (note that this class is using
// atypical notation where m31 corresponds to third column and first row, and
// code also uses "row" to mean "column" as it is usually used with matrices).
//
// This corresponds to the matrix above:
// [ m11 m21 m31 ]
// | m12 m22 m32 |
// [ m13 m23 m33 ]
//
// Now, if cos(B) != 0, C is easily derived from m11, m21, and A is equally
// easily derived from m32 and m33:
//
//  m32 / m33 = (-sin(A) * cos(B)) / (cos(A) * cos(B))
// -m32 / m33 = sin(A) / cos(A)
// -m32 / m33 = tan(A)
// => A = atan2(-m32, m33)
//
// And similarly for C.
//
// If cos(B) = 0, things get more interesting:
//
// let b = sin(B) = +-1
//
// Let's handle cases where b = 1 and b = -1 separately.
//
// b = 1
// ============================================================================
// m12 + m23 = sin(A) * b * cos(C) + cos(A) * sin(C) + cos(A) * b * sin(C) + sin(A) * cos(C)
// m12 + m23 = sin(A + C) + b * sin(A + C)
// m12 + m23 = (b + 1) * sin(A + C)
// => A = asin((m12 + m23) / (b + 1)) - C
//
// b = -1
// ============================================================================
// m13 + m22 = -cos(A) * b * cos(C) + sin(A) * sin(C) - sin(A) * b * sin(C) + cos(A) * cos(C)
// m13 + m22 = cos(A - C) - b * cos(A - C)
// m13 + m22 = (1 - b) * cos(A - C)
// => A = acos((m13 + m22) / (1 - b)) + C
//
// Technically, these aren't complete solutions for A because of periodicity,
// but we're only interested in one solution.
//
// As long as A is solved as above, C can be chosen arbitrarily. Proof for
// this is omitted.
//
J3DIMatrix4.prototype.decompose = function (_translate, _rotate, _scale, _skew, _perspective) {
	// Normalize the matrix.
	if (this.$matrix.m44 == 0)
		return false;

	// Gather the params
	var translate, rotate, scale, skew, perspective;

	var translate = (_translate == undefined || !("length" in _translate)) ? new J3DIVector3 : _translate;
	var rotate = (_rotate == undefined || !("length" in _rotate)) ? new J3DIVector3 : _rotate;
	var scale = (_scale == undefined || !("length" in _scale)) ? new J3DIVector3 : _scale;
	var skew = (_skew == undefined || !("length" in _skew)) ? new J3DIVector3 : _skew;
	var perspective = (_perspective == undefined || !("length" in _perspective)) ? new Array(4) : _perspective;

	var matrix = new J3DIMatrix4(this);

	matrix.divide(matrix.$matrix.m44);

	// perspectiveMatrix is used to solve for perspective, but it also provides
	// an easy way to test for singularity of the upper 3x3 component.
	var perspectiveMatrix = new J3DIMatrix4(matrix);

	perspectiveMatrix.$matrix.m14 = 0;
	perspectiveMatrix.$matrix.m24 = 0;
	perspectiveMatrix.$matrix.m34 = 0;
	perspectiveMatrix.$matrix.m44 = 1;

	if (perspectiveMatrix._determinant4x4() == 0)
		return false;

	// First, isolate perspective.
	if (matrix.$matrix.m14 != 0 || matrix.$matrix.m24 != 0 || matrix.$matrix.m34 != 0) {
		// rightHandSide is the right hand side of the equation.
		var rightHandSide = [matrix.$matrix.m14, matrix.$matrix.m24, matrix.$matrix.m34, matrix.$matrix.m44];

		// Solve the equation by inverting perspectiveMatrix and multiplying
		// rightHandSide by the inverse.
		var inversePerspectiveMatrix = new J3DIMatrix4(perspectiveMatrix);
		inversePerspectiveMatrix.invert();
		var transposedInversePerspectiveMatrix = new J3DIMatrix4(inversePerspectiveMatrix);
		transposedInversePerspectiveMatrix.transpose();
		transposedInversePerspectiveMatrix.multVecMatrix(perspective, rightHandSide);

		// Clear the perspective partition
		matrix.$matrix.m14 = matrix.$matrix.m24 = matrix.$matrix.m34 = 0
		matrix.$matrix.m44 = 1;
	} else {
		// No perspective.
		perspective[0] = perspective[1] = perspective[2] = 0;
		perspective[3] = 1;
	}

	// Next take care of translation
	translate[0] = matrix.$matrix.m41
	matrix.$matrix.m41 = 0
	translate[1] = matrix.$matrix.m42
	matrix.$matrix.m42 = 0
	translate[2] = matrix.$matrix.m43
	matrix.$matrix.m43 = 0

	// Now get scale and shear. 'row' is a 3 element array of 3 component vectors
	var row0 = new J3DIVector3(matrix.$matrix.m11, matrix.$matrix.m12, matrix.$matrix.m13);
	var row1 = new J3DIVector3(matrix.$matrix.m21, matrix.$matrix.m22, matrix.$matrix.m23);
	var row2 = new J3DIVector3(matrix.$matrix.m31, matrix.$matrix.m32, matrix.$matrix.m33);

	// Compute X scale factor and normalize first row.
	scale[0] = row0.vectorLength();
	row0.divide(scale[0]);

	// Compute XY shear factor and make 2nd row orthogonal to 1st.
	skew[0] = row0.dot(row1);
	row1.combine(row0, 1.0, -skew[0]);

	// Now, compute Y scale and normalize 2nd row.
	scale[1] = row1.vectorLength();
	row1.divide(scale[1]);
	skew[0] /= scale[1];

	// Compute XZ and YZ shears, orthogonalize 3rd row
	skew[1] = row1.dot(row2);
	row2.combine(row0, 1.0, -skew[1]);
	skew[2] = row1.dot(row2);
	row2.combine(row1, 1.0, -skew[2]);

	// Next, get Z scale and normalize 3rd row.
	scale[2] = row2.vectorLength();
	row2.divide(scale[2]);
	skew[1] /= scale[2];
	skew[2] /= scale[2];

	// At this point, the matrix (in rows) is orthonormal.
	// Check for a coordinate system flip.  If the determinant
	// is -1, then negate the matrix and the scaling factors.
	var pdum3 = new J3DIVector3(row1);
	pdum3.cross(row2);
	if (row0.dot(pdum3) < 0) {
		for (i = 0; i < 3; i++) {
			scale[i] *= -1;
			row[0][i] *= -1;
			row[1][i] *= -1;
			row[2][i] *= -1;
		}
	}

	// Now, get the rotations out
	rotate[1] = Math.asin(row2[0]);
	if (Math.cos(rotate[1]) != 0) {
		rotate[0] = Math.atan2(-row2[1], row2[2]);
		rotate[2] = Math.atan2(-row1[0], row0[0]);
	} else {
		rotate[2] = 0; // arbitrary in this case
		var b = Math.sin(rotate[1]);
		if (b < 0) {
			// b == -1
			rotate[0] = Math.acos((row0[2] + row1[1]) / (1 - b)) + rotate[2];
		} else {
			// b == 1
			rotate[0] = Math.asin((row1[2] + row0[1]) / (b + 1)) - rotate[2];
		}
	}

	// Convert rotations to degrees
	var rad2deg = 180 / Math.PI;
	rotate[0] *= rad2deg;
	rotate[1] *= rad2deg;
	rotate[2] *= rad2deg;

	return true;
}

J3DIMatrix4.prototype._determinant2x2 = function (a, b, c, d) {
	return a * d - b * c;
}

J3DIMatrix4.prototype._determinant3x3 = function (a1, a2, a3, b1, b2, b3, c1, c2, c3) {
	return a1 * this._determinant2x2(b2, b3, c2, c3) - b1 * this._determinant2x2(a2, a3, c2, c3) + c1 * this._determinant2x2(a2, a3, b2, b3);
}

J3DIMatrix4.prototype._determinant4x4 = function () {
	var a1 = this.$matrix.m11;
	var b1 = this.$matrix.m12;
	var c1 = this.$matrix.m13;
	var d1 = this.$matrix.m14;

	var a2 = this.$matrix.m21;
	var b2 = this.$matrix.m22;
	var c2 = this.$matrix.m23;
	var d2 = this.$matrix.m24;

	var a3 = this.$matrix.m31;
	var b3 = this.$matrix.m32;
	var c3 = this.$matrix.m33;
	var d3 = this.$matrix.m34;

	var a4 = this.$matrix.m41;
	var b4 = this.$matrix.m42;
	var c4 = this.$matrix.m43;
	var d4 = this.$matrix.m44;

	return a1 * this._determinant3x3(b2, b3, b4, c2, c3, c4, d2, d3, d4) - b1 * this._determinant3x3(a2, a3, a4, c2, c3, c4, d2, d3, d4) + c1 * this._determinant3x3(a2, a3, a4, b2, b3, b4, d2, d3, d4) - d1 * this._determinant3x3(a2, a3, a4, b2, b3, b4, c2, c3, c4);
}

J3DIMatrix4.prototype._makeAdjoint = function () {
	var a1 = this.$matrix.m11;
	var b1 = this.$matrix.m12;
	var c1 = this.$matrix.m13;
	var d1 = this.$matrix.m14;

	var a2 = this.$matrix.m21;
	var b2 = this.$matrix.m22;
	var c2 = this.$matrix.m23;
	var d2 = this.$matrix.m24;

	var a3 = this.$matrix.m31;
	var b3 = this.$matrix.m32;
	var c3 = this.$matrix.m33;
	var d3 = this.$matrix.m34;

	var a4 = this.$matrix.m41;
	var b4 = this.$matrix.m42;
	var c4 = this.$matrix.m43;
	var d4 = this.$matrix.m44;

	// Row column labeling reversed since we transpose rows & columns
	this.$matrix.m11 = this._determinant3x3(b2, b3, b4, c2, c3, c4, d2, d3, d4);
	this.$matrix.m21 = -this._determinant3x3(a2, a3, a4, c2, c3, c4, d2, d3, d4);
	this.$matrix.m31 = this._determinant3x3(a2, a3, a4, b2, b3, b4, d2, d3, d4);
	this.$matrix.m41 = -this._determinant3x3(a2, a3, a4, b2, b3, b4, c2, c3, c4);

	this.$matrix.m12 = -this._determinant3x3(b1, b3, b4, c1, c3, c4, d1, d3, d4);
	this.$matrix.m22 = this._determinant3x3(a1, a3, a4, c1, c3, c4, d1, d3, d4);
	this.$matrix.m32 = -this._determinant3x3(a1, a3, a4, b1, b3, b4, d1, d3, d4);
	this.$matrix.m42 = this._determinant3x3(a1, a3, a4, b1, b3, b4, c1, c3, c4);

	this.$matrix.m13 = this._determinant3x3(b1, b2, b4, c1, c2, c4, d1, d2, d4);
	this.$matrix.m23 = -this._determinant3x3(a1, a2, a4, c1, c2, c4, d1, d2, d4);
	this.$matrix.m33 = this._determinant3x3(a1, a2, a4, b1, b2, b4, d1, d2, d4);
	this.$matrix.m43 = -this._determinant3x3(a1, a2, a4, b1, b2, b4, c1, c2, c4);

	this.$matrix.m14 = -this._determinant3x3(b1, b2, b3, c1, c2, c3, d1, d2, d3);
	this.$matrix.m24 = this._determinant3x3(a1, a2, a3, c1, c2, c3, d1, d2, d3);
	this.$matrix.m34 = -this._determinant3x3(a1, a2, a3, b1, b2, b3, d1, d2, d3);
	this.$matrix.m44 = this._determinant3x3(a1, a2, a3, b1, b2, b3, c1, c2, c3);
}

//
// J3DIVector3
//
J3DIVector3 = function (x, y, z) {
	this.load(x, y, z);
}

J3DIVector3.prototype.load = function (x, y, z) {
	if (typeof x == 'object' && "length" in x) {
		this[0] = x[0];
		this[1] = x[1];
		this[2] = x[2];
	} else if (typeof x == 'number') {
		this[0] = x;
		this[1] = y;
		this[2] = z;
	} else {
		this[0] = 0;
		this[1] = 0;
		this[2] = 0;
	}
}

J3DIVector3.prototype.getAsArray = function () {
	return [this[0], this[1], this[2]];
}

J3DIVector3.prototype.getAsFloat32Array = function () {
	return new Float32Array(this.getAsArray());
}

J3DIVector3.prototype.vectorLength = function () {
	return Math.sqrt(this[0] * this[0] + this[1] * this[1] + this[2] * this[2]);
}

J3DIVector3.prototype.divide = function (divisor) {
	this[0] /= divisor;
	this[1] /= divisor;
	this[2] /= divisor;
}

J3DIVector3.prototype.cross = function (v) {
	var x = this[1] * v[2] - this[2] * v[1];
	var y = -this[0] * v[2] + this[2] * v[0];
	this[2] = this[0] * v[1] - this[1] * v[0];
	this[0] = x;
	this[1] = y;
}

J3DIVector3.prototype.dot = function (v) {
	return this[0] * v[0] + this[1] * v[1] + this[2] * v[2];
}

J3DIVector3.prototype.combine = function (v, ascl, bscl) {
	this[0] = (ascl * this[0]) + (bscl * v[0]);
	this[1] = (ascl * this[1]) + (bscl * v[1]);
	this[2] = (ascl * this[2]) + (bscl * v[2]);
}

J3DIVector3.prototype.multVecMatrix = function (matrix) {
	var x = this[0];
	var y = this[1];
	var z = this[2];

	this[0] = matrix.$matrix.m41 + x * matrix.$matrix.m11 + y * matrix.$matrix.m21 + z * matrix.$matrix.m31;
	this[1] = matrix.$matrix.m42 + x * matrix.$matrix.m12 + y * matrix.$matrix.m22 + z * matrix.$matrix.m32;
	this[2] = matrix.$matrix.m43 + x * matrix.$matrix.m13 + y * matrix.$matrix.m23 + z * matrix.$matrix.m33;
	var w = matrix.$matrix.m44 + x * matrix.$matrix.m14 + y * matrix.$matrix.m24 + z * matrix.$matrix.m34;
	if (w != 1 && w != 0) {
		this[0] /= w;
		this[1] /= w;
		this[2] /= w;
	}
}

J3DIVector3.prototype.toString = function () {
	return "[" + this[0] + "," + this[1] + "," + this[2] + "]";
}