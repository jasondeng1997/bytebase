INSERT INTO
    instance (
        id,
        creator_id,
        updater_id,
        environment_id,
        name,
        `engine`,
        host,
        port,
        external_link
    )
VALUES
    (
        6001,
        101,
        101,
        5001,
        'On-premises Dev MySQL',
        'MYSQL',
        'mysql.dev.example.com',
        '3306',
        'bytebase.com'
    );

INSERT INTO
    instance (
        id,
        creator_id,
        updater_id,
        environment_id,
        name,
        `engine`,
        host,
        port,
        external_link
    )
VALUES
    (
        6002,
        101,
        101,
        5002,
        'On-premises Integration MySQL',
        'MYSQL',
        'mysql.integration.example.com',
        '3306',
        'bytebase.com'
    );

INSERT INTO
    instance (
        id,
        creator_id,
        updater_id,
        environment_id,
        name,
        `engine`,
        host,
        port,
        external_link
    )
VALUES
    (
        6003,
        101,
        101,
        5003,
        'On-premises Staging MySQL',
        'MYSQL',
        'mysql.staging.example.com',
        '3306',
        'bytebase.com'
    );

INSERT INTO
    instance (
        id,
        creator_id,
        updater_id,
        environment_id,
        name,
        `engine`,
        host,
        port,
        external_link
    )
VALUES
    (
        6004,
        101,
        101,
        5004,
        'On-premises Prod MySQL',
        'MYSQL',
        'mysql.prod.example.com',
        '3306',
        'bytebase.com'
    );