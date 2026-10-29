#!/usr/bin/env python3
import json
import os
import shutil
import sqlite3
import tempfile
from pathlib import Path
from urllib.parse import unquote, urlparse
from urllib.request import Request, urlopen

ROOT = Path(__file__).resolve().parent
DB_PATH = ROOT / "serifhealth.db"
JSON_PATHS = (
    ROOT / "dataInput" / "2026-08-31_centene-management-company-llc_fidelis-ex_in-network.json",
    ROOT / "dataInput" / "2026-08-31_centene-management-company-llc_fidelis-es_in-network.json",
)
INDEX_PATH = ROOT / "dataInput" / "ParentFile" / "2026-08-31_fidelis_index.json"


def to_json(value):
    return json.dumps(value, separators=(",", ":"))


def download_missing_json_files():
    with INDEX_PATH.open("r", encoding="utf-8") as fh:
        index = json.load(fh)

    urls_by_name = {}
    for structure in index.get("reporting_structure", []):
        for file_info in structure.get("in_network_files", []):
            location = file_info.get("location", "")
            file_name = Path(unquote(urlparse(location).path)).name
            if file_name:
                urls_by_name[file_name] = location

    for json_path in JSON_PATHS:
        if json_path.exists() and json_path.stat().st_size > 0:
            continue

        url = urls_by_name.get(json_path.name)
        if not url:
            raise ValueError(f"No download URL found in the index for {json_path.name}")

        temporary_path = json_path.with_name(f".{json_path.name}.part")
        try:
            request = Request(url, headers={"User-Agent": "SerifHealthImporter/1.0"})
            with urlopen(request, timeout=120) as response, temporary_path.open("wb") as target:
                shutil.copyfileobj(response, target, length=1024 * 1024)
            if temporary_path.stat().st_size == 0:
                raise ValueError("The downloaded file is empty")
            os.replace(temporary_path, json_path)
            print(f"Downloaded {json_path.name}")
        except Exception as err:
            temporary_path.unlink(missing_ok=True)
            raise RuntimeError(f"Failed to download {json_path.name} from the parent index") from err


def create_schema(conn):
    conn.executescript(
        """
        PRAGMA foreign_keys = OFF;
        
        DROP TABLE IF EXISTS in_network_file;
        DROP TABLE IF EXISTS reporting_plan;
        DROP TABLE IF EXISTS reporting_plan_in_network_file;
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

        CREATE TABLE reporting_plan (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            plan_name TEXT,
            plan_id_type TEXT,
            plan_id TEXT,
            plan_market_type TEXT,
            issuer_name TEXT
        );

        CREATE TABLE in_network_file (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            description TEXT,
            file_name TEXT NOT NULL UNIQUE
        );

        CREATE TABLE reporting_plan_in_network_file (
            reporting_plan_id INTEGER NOT NULL,
            in_network_file_id INTEGER NOT NULL,
            PRIMARY KEY (reporting_plan_id, in_network_file_id),
            FOREIGN KEY (reporting_plan_id) REFERENCES reporting_plan(id),
            FOREIGN KEY (in_network_file_id) REFERENCES in_network_file(id)
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
            in_network_file_id INTEGER NOT NULL,
            negotiation_arrangement TEXT,
            name TEXT,
            billing_code_type TEXT,
            billing_code_type_version TEXT,
            billing_code TEXT,
            description TEXT,
            FOREIGN KEY (reporting_entity_id) REFERENCES reporting_entity(id),
            FOREIGN KEY (in_network_file_id) REFERENCES in_network_file(id)
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
        CREATE INDEX idx_network_item_file_id ON network_item (in_network_file_id);
        CREATE INDEX idx_reporting_plan_file_id ON reporting_plan_in_network_file (in_network_file_id);
        CREATE INDEX idx_negotiated_price_setting ON negotiated_price (setting);
        CREATE INDEX idx_provider_reference_group_id ON provider_reference (provider_group_id);
        CREATE INDEX idx_provider_group_reference_id ON provider_group (provider_reference_id);

        PRAGMA foreign_keys = ON;
        """
    )


def import_index(cur):
    with INDEX_PATH.open("r", encoding="utf-8") as fh:
        index = json.load(fh)

    configured_files = {path.name for path in JSON_PATHS}
    file_ids = {}
    file_id_by_name = {}
    for structure in index.get("reporting_structure", []):
        structure_file_ids = []
        for file_info in structure.get("in_network_files", []):
            location = file_info.get("location", "")
            file_name = Path(unquote(urlparse(location).path)).name
            if not file_name:
                raise ValueError("An in_network_file entry has no filename in its location")
            if file_name not in configured_files:
                raise ValueError(f"No input JSON is configured for indexed file {file_name}")

            if file_name not in file_ids:
                file_ids[file_name] = cur.execute(
                    "INSERT INTO in_network_file (description, file_name) VALUES (?, ?)",
                    (file_info.get("description"), file_name),
                ).lastrowid
            file_id = file_ids[file_name]
            file_id_by_name[file_name] = file_id
            structure_file_ids.append(file_id)

        for plan in structure.get("reporting_plans", []):
            plan_id = cur.execute(
                """
                INSERT INTO reporting_plan (
                    plan_name, plan_id_type, plan_id, plan_market_type, issuer_name
                ) VALUES (?, ?, ?, ?, ?)
                """,
                (
                    plan.get("plan_name"),
                    plan.get("plan_id_type"),
                    plan.get("plan_id"),
                    plan.get("plan_market_type"),
                    plan.get("issuer_name"),
                ),
            ).lastrowid
            for file_id in structure_file_ids:
                cur.execute(
                    "INSERT OR IGNORE INTO reporting_plan_in_network_file "
                    "(reporting_plan_id, in_network_file_id) VALUES (?, ?)",
                    (plan_id, file_id),
                )

    missing_mappings = [path.name for path in JSON_PATHS if path.name not in file_id_by_name]
    if missing_mappings:
        raise ValueError(f"No in_network_files association found for {missing_mappings[0]}")
    return file_id_by_name


def import_payload(cur, json_path, in_network_file_id, next_item_id, next_provider_reference_id):
    with json_path.open("r", encoding="utf-8") as fh:
        payload = json.load(fh)

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

    provider_id_map = {}
    for provider_ref in payload.get("provider_references", []):
        source_provider_id = provider_ref.get("provider_group_id")
        if source_provider_id in provider_id_map:
            raise ValueError(f"Duplicate provider reference {source_provider_id} in {json_path.name}")

        provider_id = next_provider_reference_id
        next_provider_reference_id += 1
        provider_id_map[source_provider_id] = provider_id
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
                source_provider_id,
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

    for item in payload.get("in_network", []):
        network_item_id = next_item_id
        next_item_id += 1
        cur.execute(
            """
            INSERT INTO network_item (
                id,
                reporting_entity_id,
                in_network_file_id,
                negotiation_arrangement,
                name,
                billing_code_type,
                billing_code_type_version,
                billing_code,
                description
            ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
            """,
            (
                network_item_id,
                reporting_entity_id,
                in_network_file_id,
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
            source_provider_ids = rate.get("provider_references", [])
            try:
                provider_ids = [provider_id_map[source_id] for source_id in source_provider_ids]
            except KeyError as err:
                raise ValueError(
                    f"Unknown provider reference {err.args[0]} in {json_path.name} "
                    f"for billing code {item.get('billing_code')}"
                ) from err

            for price in rate.get("negotiated_prices", []):
                cur.execute(
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
                        to_json(provider_ids),
                    ),
                )

            for provider_reference_id in provider_ids:
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

    return next_item_id, next_provider_reference_id, len(payload.get("in_network", [])), len(provider_id_map)


def main():
    download_missing_json_files()
    missing_files = [path for path in JSON_PATHS if not path.exists()]
    if missing_files:
        raise FileNotFoundError(f"Data file not found: {missing_files[0]}")

    fd, temporary_path = tempfile.mkstemp(prefix="serifhealth-", suffix=".db", dir=ROOT)
    os.close(fd)
    temporary_path = Path(temporary_path)
    conn = None
    try:
        conn = sqlite3.connect(temporary_path)
        create_schema(conn)
        cur = conn.cursor()
        file_id_by_name = import_index(cur)
        next_item_id = 1
        next_provider_reference_id = 1
        total_items = 0
        total_provider_references = 0

        for json_path in JSON_PATHS:
            (
                next_item_id,
                next_provider_reference_id,
                item_count,
                provider_reference_count,
            ) = import_payload(
                cur,
                json_path,
                file_id_by_name[json_path.name],
                next_item_id,
                next_provider_reference_id,
            )
            conn.commit()
            total_items += item_count
            total_provider_references += provider_reference_count
            print(f"Imported {item_count} network items from {json_path.name}")

        conn.close()
        conn = None
        os.replace(temporary_path, DB_PATH)
    except Exception:
        if conn is not None:
            conn.close()
        temporary_path.unlink(missing_ok=True)
        raise

    print(f"SQLite database created at: {DB_PATH}")
    print(f"Rows imported: {total_items} network items, {total_provider_references} provider references")


if __name__ == "__main__":
    main()
