#!/usr/bin/env python2

#
# This takes the output of extract_nexrad and generates images.
# It isn't very smart at the moment and won't draw anything
# until all input has been consumed, so it's not very useful
# for realtime continuous generation of maps.
#

import sys, math
import cairo, colorsys

intensities = {
    0: colorsys.hls_to_rgb(240.0/360.0, 0.15, 1.0),
    1: colorsys.hls_to_rgb(240.0/360.0, 0.2, 1.0),
    2: colorsys.hls_to_rgb(200.0/360.0, 0.4, 1.0),
    3: colorsys.hls_to_rgb(160.0/360.0, 0.4, 1.0),
    4: colorsys.hls_to_rgb(120.0/360.0, 0.5, 1.0),
    5: colorsys.hls_to_rgb(80.0/360.0, 0.5, 1.0),
    6: colorsys.hls_to_rgb(40.0/360.0, 0.6, 1.0),
    7: colorsys.hls_to_rgb(0.0/360.0, 0.7, 1.0)
}

def color_for(intensity):    
    r,g,b = intensities[intensity]
    return cairo.SolidPattern(r,g,b,1.0)

# mercator projection (yeah, it's not great, but it's simple)
# lat, lon are in _arcminutes_
def project(lat,lon):
    lat /= 60.0
    lat = math.pi * lat / 180.0

    lon /= 60.0
    lon -= 360.0
    lon = math.pi * lon / 180.0

    x = lon
    y = math.log(math.tan(math.pi/4.0 + lat/2.0))

    return (x,y)

images = {}

while True:
    line = sys.stdin.readline()
    if not line: break

    words = line.strip().split(' ')
    if words[0] != 'NEXRAD': continue

    nexrad, maptype, maptime, sf, latN, lonW, latSize, lonSize, blockdata = words
    sf = int(sf)
    latN = int(latN)
    lonW = int(lonW)
    latSize = int(latSize)
    lonSize = int(lonSize)

    key = maptype + '/' + maptime
    if not key in images:
        images[key] = {
            'type' : maptype,
            'time' : maptime,
            'lat_min' : latN - latSize,
            'lat_max' : latN,
            'lon_min' : lonW,
            'lon_max' : lonW + lonSize,
            'blocks' : {
                sf : [ (latN, lonW, latSize, lonSize, blockdata) ]
            }
        }
    else:
        image = images[key]
        image['lat_min'] = min(image['lat_min'], latN - latSize)
        image['lat_max'] = max(image['lat_max'], latN)
        image['lon_min'] = min(image['lon_min'], lonW)
        image['lon_max'] = max(image['lon_max'], lonW + lonSize)

        if not sf in image['blocks']:
            image['blocks'][sf] = [ (latN, lonW, latSize, lonSize, blockdata) ]
        else:
            image['blocks'][sf].append( (latN, lonW, latSize, lonSize, blockdata) )

for image in images.values():
    lat_min = image['lat_min']
    lat_max = image['lat_max']
    lon_min = image['lon_min']
    lon_max = image['lon_max']

    # find most detailed scale; scale our image accordingly
    sf = min(image['blocks'].keys())
    if sf == 1: scale = 5.0
    elif sf == 2: scale = 9.0
    else: scale = 1.0
    pixels_per_degree = 80.0 / scale

    # project, find scale
    x0,y0 = project(lat_min,lon_min)
    x1,y1 = project(lat_min,lon_max)
    x2,y2 = project(lat_max,lon_min)
    x3,y3 = project(lat_max,lon_max)
    xmin = min(x0,x1,x2,x3)
    xmax = max(x0,x1,x2,x3)
    ymin = min(y0,y1,y2,y3)
    ymax = max(y0,y1,y2,y3)

    xsize = int(pixels_per_degree * 180.0 * (xmax - xmin) / math.pi)
    ysize = int(pixels_per_degree * 180.0 * (ymax - ymin) / math.pi)

    print image['type'], image['time'], 'dimensions', xsize, ysize, 'layers', len(image['blocks'])

    surface = cairo.ImageSurface(cairo.FORMAT_RGB24, xsize, ysize)
    cc = cairo.Context(surface)
    cc.set_antialias(cairo.ANTIALIAS_NONE)

    cc.scale(1.0 * xsize / (xmax - xmin), -1.0 * ysize / (ymax - ymin))
    cc.translate(-xmin, -ymax)

    if image['type'] == 'CONUS':
        cc.set_source(color_for(0))
    else:
        r,g,b = colorsys.hls_to_rgb(270.0/360.0, 0.10, 1.0)
        cc.set_source(cairo.SolidPattern(r,g,b,1.0))

    cc.paint()
    
    for sf in sorted(image['blocks'].keys(), reverse=True): # lowest res first    
        for latN, lonW, latSize, lonSize, data in image['blocks'][sf]:
            for y in xrange(4):
                for x in xrange(32):
                    intensity = int(data[x+y*32])
                    lat = latN - y * latSize / 4.0
                    lon = lonW + x * lonSize / 32.0
                
                    cc.move_to(*project(lat,lon))
                    cc.line_to(*project(lat-latSize/4.0,lon))
                    cc.line_to(*project(lat-latSize/4.0,lon+lonSize/32.0))
                    cc.line_to(*project(lat,lon+lonSize/32.0))
                    cc.close_path()
                    cc.set_source(color_for(intensity))
                    cc.fill()

    surface.write_to_png('nexrad_%s_%s.png' % (image['type'], image['time']))


    
