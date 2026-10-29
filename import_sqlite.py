#!/usr/bin/env python3
import json
import sqlite3
from pathlib import Path

ROOT = Path(__file__).resolve().parent
DB_PATH = ROOT / "serifhealth.db"
JSON_PATH = ROOT / "dataInput" / "2026-08-31_centene-management-company-llc_fidelis-ex_in-network.json"


def to_json(value):
    return json.dumps(value, separators=(",", ":"))


def create_schema(conn):
    conn.executescript(
        """
        PRAGMA foreign_keys = OFF;

        DROP TABLE IF EXISTS reporting_entity;
        DROP TABLE IF EXISTS network_item_provider_reference;
        DROP TABLE IF EXISTS negotiated_price;
        DROP TABLE IF EXISTS network_item;
        DROP TABLE IF EXISTS provider_group;
        DROP TABLE IF EXISTS provider_reference;

        CREATE TABLE reporting_entity (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            name TEXT NOT NULL,
            entity_type TEXT,
            version TEXT,
            last_updated_on TEXT
        );

        CREATE TABLE provider_reference (
            id INTEGER PRIMARY KEY,
            provider_group_id INTEGER,
            network_name TEXT
        );

        CREATE TABLE provider_group (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            provider_reference_id INTEGER NOT NULL,
            npi TEXT,
            tin_type TEXT,
            tin_value TEXT,
            tin_business_name TEXT,
            FOREIGN KEY (provider_reference_id) REFERENCES provider_reference(id)
        );

        CREATE TABLE network_item (
            id INTEGER PRIMARY KEY,
            reporting_entity_id INTEGER NOT NULL,
            negotiation_arrangement TEXT,
            name TEXT,
            billing_code_type TEXT,
            billing_code_type_version TEXT,
            billing_code TEXT,
            description TEXT,
            FOREIGN KEY (reporting_entity_id) REFERENCES reporting_entity(id)
        );

        CREATE TABLE negotiated_price (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            network_item_id INTEGER NOT NULL,
            negotiated_type TEXT,
            negotiated_rate REAL,
            expiration_date TEXT,
            service_code TEXT,
            billing_class TEXT,
            setting TEXT,
            billing_code_modifier TEXT,
            provider_references TEXT,
            FOREIGN KEY (network_item_id) REFERENCES network_item(id)
        );

        CREATE TABLE network_item_provider_reference (
            network_item_id INTEGER NOT NULL,
            provider_reference_id INTEGER NOT NULL,
            PRIMARY KEY (network_item_id, provider_reference_id),
            FOREIGN KEY (network_item_id) REFERENCES network_item(id),
            FOREIGN KEY (provider_reference_id) REFERENCES provider_reference(id)
        );

        CREATE INDEX idx_network_item_billing_code ON network_item (billing_code);
        CREATE INDEX idx_negotiated_price_setting ON negotiated_price (setting);
        CREATE INDEX idx_provider_reference_group_id ON provider_reference (provider_group_id);
        CREATE INDEX idx_provider_group_reference_id ON provider_group (provider_reference_id);

        PRAGMA foreign_keys = ON;
        """
    )


def main():
    if not JSON_PATH.exists():
        raise FileNotFoundError(f"Data file not found: {JSON_PATH}")

    with JSON_PATH.open("r", encoding="utf-8") as fh:
        payload = json.load(fh)

    conn = sqlite3.connect(DB_PATH)
    conn.row_factory = sqlite3.Row
    create_schema(conn)
    cur = conn.cursor()

    reporting_entity_id = cur.execute(
        """
        INSERT INTO reporting_entity (name, entity_type, version, last_updated_on)
        VALUES (?, ?, ?, ?)
        """,
        (
            payload.get("reporting_entity_name"),
            payload.get("reporting_entity_type"),
            payload.get("version"),
            payload.get("last_updated_on"),
        ),
    ).lastrowid

    for provider_ref in payload.get("provider_references", []):
        provider_id = provider_ref.get("provider_group_id")
        cur.execute(
            """
            INSERT INTO provider_reference (
                id,
                provider_group_id,
                network_name
            ) VALUES (?, ?, ?)
            """,
            (
                provider_id,
                provider_id,
                to_json(provider_ref.get("network_name", [])),
            ),
        )

        for group in provider_ref.get("provider_groups", []):
            tin = group.get("tin") or {}
            cur.execute(
                """
                INSERT INTO provider_group (
                    provider_reference_id,
                    npi,
                    tin_type,
                    tin_value,
                    tin_business_name
                ) VALUES (?, ?, ?, ?, ?)
                """,
                (
                    provider_id,
                    to_json(group.get("npi", [])),
                    tin.get("type"),
                    tin.get("value"),
                    tin.get("business_name"),
                ),
            )

    for idx, item in enumerate(payload.get("in_network", []), start=1):
        network_item_id = cur.execute(
            """
            INSERT INTO network_item (
                id,
                reporting_entity_id,
                negotiation_arrangement,
                name,
                billing_code_type,
                billing_code_type_version,
                billing_code,
                description
            ) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
            """,
            (
                idx,
                reporting_entity_id,
                item.get("negotiation_arrangement"),
                item.get("name"),
                item.get("billing_code_type"),
                item.get("billing_code_type_version"),
                item.get("billing_code"),
                item.get("description"),
            ),
        ).lastrowid

        seen_provider_links = set()
        for rate in item.get("negotiated_rates", []):
            for price in rate.get("negotiated_prices", []):
                price_id = cur.execute(
                    """
                    INSERT INTO negotiated_price (
                        network_item_id,
                        negotiated_type,
                        negotiated_rate,
                        expiration_date,
                        service_code,
                        billing_class,
                        setting,
                        billing_code_modifier,
                        provider_references
                    ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
                    """,
                    (
                        network_item_id,
                        price.get("negotiated_type"),
                        price.get("negotiated_rate"),
                        price.get("expiration_date"),
                        to_json(price.get("service_code", [])),
                        price.get("billing_class"),
                        price.get("setting"),
                        to_json(price.get("billing_code_modifier", [])),
                        to_json(rate.get("provider_references", [])),
                    ),
                ).lastrowid

                for provider_reference_id in rate.get("provider_references", []):
                    if (network_item_id, provider_reference_id) not in seen_provider_links:
                        seen_provider_links.add((network_item_id, provider_reference_id))
                        cur.execute(
                            """
                            INSERT OR IGNORE INTO network_item_provider_reference (
                                network_item_id,
                                provider_reference_id
                            ) VALUES (?, ?)
                            """,
                            (network_item_id, provider_reference_id),
                        )

    conn.commit()
    conn.close()

    print(f"SQLite database created at: {DB_PATH}")
    print(f"Rows imported: {len(payload.get('in_network', []))} network items, {len(payload.get('provider_references', []))} provider references")


if __name__ == "__main__":
    main()
