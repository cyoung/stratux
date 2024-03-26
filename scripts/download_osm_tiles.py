#!/usr/bin/python3

from sys import argv
import os
import math
import urllib.request
import random
import os.path
import sqlite3


URL_TEMPLATE = "https://c.tile.openstreetmap.org/%d/%d/%d.png"
BBOX = None # [lon_min, lat_min, lon_max, lat_max] or None for whole world
ZOOM_MAX = 7
LAYERTYPE = "baselayer" # "baselayer" or "overlay"
LAYERNAME = "OSM Low Detail"
TILE_FORMAT = "png"

def deg2num(lat_deg, lon_deg, zoom):
    lat_rad = math.radians(lat_deg)
    n = 2.0 ** zoom
    xtile = int((lon_deg + 180.0) / 360.0 * n)
    ytile = int((1.0 - math.log(math.tan(lat_rad) + (1 / math.cos(lat_rad))) / math.pi) / 2.0 * n)
    return (xtile, ytile)

def download_url(zoom, xtile, ytile, cursor):
    subdomain = random.randint(1, 4)
    
    url = URL_TEMPLATE % (zoom, xtile, ytile)

    ymax = 1 << zoom
    yinverted = ymax - ytile - 1

    existing = cursor.execute('SELECT count(*) FROM tiles WHERE zoom_level=? AND tile_column=? AND tile_row=?', (zoom, xtile, yinverted)).fetchall()
    if existing[0][0] > 0:
        print('Skipping ' + url)
        return
    

    print("downloading %r" % url)
    request = urllib.request.Request(
        url, data=None,
        headers={
            'User-Agent': 'Low-Zoom Downloader'
        }
    )
    source = urllib.request.urlopen(request)
    content = source.read()
    source.close()
    cursor.execute('INSERT INTO tiles(zoom_level, tile_column, tile_row, tile_data) VALUES(?, ?, ?, ?)', (zoom, xtile, yinverted, content))


def main(argv):

    db = argv[1] if len(argv) > 1 else 'osm.mbtiles'
    conn = sqlite3.connect(db)
    cur = conn.cursor()
    bboxStr = "-180,-85,180,85" if BBOX is None else ",".join(map(str, BBOX))

    cur.executescript('''
    CREATE TABLE IF NOT EXISTS tiles (
            zoom_level integer,
            tile_column integer,
            tile_row integer,
            tile_data blob);
    CREATE TABLE IF NOT EXISTS metadata(name text, value text);
    CREATE UNIQUE INDEX IF NOT EXISTS metadata_name on metadata (name);
    CREATE UNIQUE INDEX IF NOT EXISTS tile_index on tiles(zoom_level, tile_column, tile_row);
    INSERT OR REPLACE INTO metadata VALUES('minzoom', '1');
    INSERT OR REPLACE INTO metadata VALUES('maxzoom', '{0}');
    INSERT OR REPLACE INTO metadata VALUES('name', '{1}');
    INSERT OR REPLACE INTO metadata VALUES('type', '{2}');
    INSERT OR REPLACE INTO metadata VALUES('format', '{3}');
    INSERT OR REPLACE INTO metadata VALUES('bounds', '{4}');

    '''.format(ZOOM_MAX, LAYERNAME, LAYERTYPE, TILE_FORMAT, bboxStr))

    # from 0 to 6 download all
    for zoom in range(0, ZOOM_MAX+1):
        xstart = 0
        ystart = 0
        xend = 2**zoom-1
        yend = 2**zoom-1
        if BBOX is not None:
            xstart, yend = deg2num(BBOX[1], BBOX[0], zoom)
            xend, ystart = deg2num(BBOX[3], BBOX[2], zoom)

        for x in range(xstart, xend+1):
            for y in range(ystart, yend+1):
                download_url(zoom, x, y, cur)
            
            conn.commit()

    
    cur.close()
    conn.close()

main(argv)

