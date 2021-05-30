#!/usr/bin/python3

from sys import argv
import os
import math
import urllib.request
import random
import os.path
import sqlite3

def deg2num(lat_deg, lon_deg, zoom):
    lat_rad = math.radians(lat_deg)
    n = 2.0 ** zoom
    xtile = int((lon_deg + 180.0) / 360.0 * n)
    ytile = int((1.0 - math.log(math.tan(lat_rad) + (1 / math.cos(lat_rad))) / math.pi) / 2.0 * n)
    return (xtile, ytile)

def download_url(zoom, xtile, ytile, cursor):
    subdomain = random.randint(1, 4)
    
    url = "https://c.tile.openstreetmap.org/%d/%d/%d.png" % (zoom, xtile, ytile)
    download_path = "tiles/%d/%d/%d.png" % (zoom, xtile, ytile)

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
    maxzoom = 7 # redefine me if need so

    db = argv[1] if len(argv) > 1 else 'osm.mbtiles'
    conn = sqlite3.connect(db)
    cur = conn.cursor()
    cur.executescript('''
    CREATE TABLE IF NOT EXISTS tiles (
            zoom_level integer,
            tile_column integer,
            tile_row integer,
            tile_data blob);
    CREATE TABLE IF NOT EXISTS metadata(name text, value text);
    CREATE UNIQUE INDEX IF NOT EXISTS metadata_name on metadata (name);
    CREATE UNIQUE INDEX IF NOT EXISTS tile_index on tiles(zoom_level, tile_column, tile_row);
    INSERT OR REPLACE INTO metadata VALUES('type', 'baselayer');
    INSERT OR REPLACE INTO metadata VALUES('format', 'png');
    INSERT OR REPLACE INTO metadata VALUES('minzoom', '1');
    INSERT OR REPLACE INTO metadata VALUES('maxzoom', '{0}');
    INSERT OR REPLACE INTO metadata VALUES('bounds', '-180,-85,180,85');
    '''.format(maxzoom))

    # from 0 to 6 download all
    for zoom in range(0,maxzoom+1,1):
        for x in range(0,2**zoom,1):
            for y in range(0,2**zoom,1):
                download_url(zoom, x, y, cur)
    
    cur.close()
    conn.commit()
    conn.close()

main(argv)    
