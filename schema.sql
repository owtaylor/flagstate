DROP TABLE IF EXISTS image, image_tag, list, list_tag, list_entry CASCADE;

CREATE TABLE modification (
       modification_time timestamp with time zone
);
INSERT INTO modification VALUES (now());

CREATE TABLE image (
       digest text PRIMARY KEY,
       media_type text,
       arch text,
       os text,
       annotations jsonb
);
CREATE INDEX image_annotations ON image USING gin(annotations);

CREATE TABLE image_tag (
       repository text,
       tag text,
       image text REFERENCES image(digest)
);
CREATE UNIQUE INDEX image_tag_pkey ON image_tag ( repository, tag );

CREATE TABLE list (
       digest text PRIMARY KEY,
       media_type text,
       annotations jsonb
);
CREATE INDEX list_annotations ON list USING gin(annotations);

CREATE TABLE list_tag (
       repository text,
       tag text,
       list text REFERENCES list(digest)
);
CREATE UNIQUE INDEX list_tag_pkey ON list_tag ( repository, tag );

CREATE TABLE list_entry (
       list text REFERENCES list(digest) ON DELETE CASCADE,
       image text REFERENCES image(digest)
);
CREATE UNIQUE INDEX list_entry_pkey ON list_entry ( list, image );
