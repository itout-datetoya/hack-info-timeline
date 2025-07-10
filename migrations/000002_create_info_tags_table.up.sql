CREATE TABLE hacking_info_tags (
    info_id BIGINT NOT NULL REFERENCES hacking_infos(id) ON DELETE CASCADE,
    tag_id BIGINT NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (info_id, tag_id)
);

CREATE TABLE transfer_info_tags (
    info_id BIGINT NOT NULL REFERENCES transfer_infos(id) ON DELETE CASCADE,
    tag_id BIGINT NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (info_id, tag_id)
);