DROP TABLE IF EXISTS modification, image, imageTag, list, listTag, listEntry CASCADE;

CREATE TABLE modification (
       ModificationTime timestamp with time zone
);
INSERT INTO modification VALUES (now());

CREATE TABLE image (
       Digest text PRIMARY KEY,
       MediaType text,
       Arch text,
       OS text,
       Annotations jsonb
);
CREATE INDEX imageAnnotations ON image USING gin(Annotations);

CREATE TABLE imageTag (
       Repository text,
       Tag text,
       Image text REFERENCES image(Digest)
);
CREATE UNIQUE INDEX imageTagPKey ON imageTag ( Repository, Tag );
CREATE INDEX imageTagTag ON imageTag ( Tag );

CREATE TABLE list (
       Digest text PRIMARY KEY,
       MediaType text,
       Annotations jsonb
);
CREATE INDEX listAnnotations ON list USING gin(Annotations);

CREATE TABLE listTag (
       Repository text,
       Tag text,
       List text REFERENCES list(Digest)
);
CREATE UNIQUE INDEX listTagPKey ON listTag ( Repository, Tag );
CREATE INDEX listTagTag ON listTag ( Tag );

CREATE TABLE listEntry (
       List text REFERENCES list(Digest) ON DELETE CASCADE,
       Image text REFERENCES image(Digest)
);
CREATE UNIQUE INDEX listEntryPKey ON listEntry ( List, Image );
