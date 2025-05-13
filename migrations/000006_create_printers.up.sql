ALTER TABLE stations DROP CONSTRAINT IF EXISTS fk_stations_printer;
ALTER TABLE stations DROP CONSTRAINT IF EXISTS fk_stations_display;
DROP TABLE IF EXISTS printers;
DROP TABLE IF EXISTS displays;