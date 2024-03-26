#!/bin/bash

mkdir -p tmp
cd tmp


# OFM.. already mbtiles. THe tiles are already nicely aligned - existing tiles = filled tiles.
# So we can just merge all DBs together
AIRAC=2307
for f in ebbu ed efin ehaa ekdk epww esaa lbsr ldzo lf lggg lhcc li ljla lkaa lovv lrbb lsas lzbb; do
    wget https://snapshots.openflightmaps.org/live/${AIRAC}/tiles/${f}/noninteractive/epsg3857/${f}_256.mbtiles
done

sqlite3 out.db <<EOF
CREATE TABLE tiles (
            zoom_level integer,
            tile_column integer,
            tile_row integer,
            tile_data blob);
CREATE TABLE metadata
        (name text, value text);
CREATE TABLE grids (zoom_level integer, tile_column integer,
    tile_row integer, grid blob);
CREATE TABLE grid_data (zoom_level integer, tile_column
    integer, tile_row integer, key_name text, key_json text);
CREATE UNIQUE INDEX name on metadata (name);
CREATE UNIQUE INDEX tile_index on tiles
        (zoom_level, tile_column, tile_row);
INSERT INTO metadata VALUES('name', 'OpenFlightMap Europe');
INSERT INTO metadata VALUES('type', 'baselayer');
INSERT INTO metadata VALUES('format', 'png');
INSERT INTO metadata VALUES('minzoom', '4');
INSERT INTO metadata VALUES('maxzoom', '11');
EOF

for f in *.mbtiles; do
    echo "Adding $f"
    sqlite3 out.db "ATTACH DATABASE '$f' AS indb; INSERT OR IGNORE INTO tiles SELECT * FROM indb.tiles"
done
mv out.db ../openflightmaps.mbtiles
cd ..
rm -r tmp
