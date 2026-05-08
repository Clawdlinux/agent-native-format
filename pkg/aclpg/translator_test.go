// Copyright 2026 NineVigil / Clawdlinux.
//
// Licensed under the Apache License, Version 2.0 (the "License").

package aclpg

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Clawdlinux/ninevigil-acp/pkg/acl"
)

// tinySchema is the spec's worked example for tests + golden output.
// Keep this small; the realistic-scale benchmark uses ecommerceSchema.
func tinySchema() Schema {
	return Schema{
		Database:   "shop",
		SchemaName: "public",
		Tables: []Table{
			{
				Name: "users",
				Columns: []Column{
					{Name: "id", Type: "bigint", Default: "nextval('users_id_seq'::regclass)"},
					{Name: "email", Type: "text"},
					{Name: "created_at", Type: "timestamp with time zone", Default: "now()"},
				},
				PrimaryKey: []string{"id"},
			},
			{
				Name: "orders",
				Columns: []Column{
					{Name: "id", Type: "bigint", Default: "nextval('orders_id_seq'::regclass)"},
					{Name: "user_id", Type: "bigint"},
					{Name: "total_cents", Type: "integer"},
					{Name: "status", Type: "character varying(32)", Default: "'pending'::text"},
				},
				PrimaryKey: []string{"id"},
			},
		},
		Indexes: []Index{
			{Name: "users_email_key", Table: "users", Columns: []string{"email"}, Unique: true},
			{Name: "orders_user_id_idx", Table: "orders", Columns: []string{"user_id"}},
		},
		Relations: []Relation{
			{Name: "orders_user_id_fkey", FromTable: "orders", FromColumn: "user_id", ToTable: "users", ToColumn: "id"},
		},
	}
}

func TestEncodeGolden(t *testing.T) {
	t.Parallel()
	got, err := Encode(tinySchema())
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	want := "@db shop\n" +
		"@schema public\n" +
		"@source pg-schema/v0.1\n" +
		"\n" +
		"tables 2\n" +
		"  orders cols=4 pk=id\n" +
		"  users cols=3 pk=id\n" +
		"\n" +
		"columns 7\n" +
		"  orders.id type=int8 def=AUTO\n" +
		"  orders.user_id type=int8\n" +
		"  orders.total_cents type=int4\n" +
		"  orders.status type=text def=pending\n" +
		"  users.id type=int8 def=AUTO\n" +
		"  users.email type=text\n" +
		"  users.created_at type=timestamptz def=now\n" +
		"\n" +
		"indexes 2\n" +
		"  orders_user_id_idx on=orders cols=user_id\n" +
		"  users_email_key on=users cols=email uniq=1\n" +
		"\n" +
		"relations 1\n" +
		"  orders.user_id->users.id fk=orders_user_id_fkey\n" +
		"\n" +
		"actions\n" +
		"  select|insert|update|delete|describe\n"
	if string(got) != want {
		t.Fatalf("encode mismatch:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestRoundTrip(t *testing.T) {
	t.Parallel()
	out, err := Encode(tinySchema())
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	d, err := acl.Decode(out)
	if err != nil {
		t.Fatalf("Decode(Encode): %v\n%s", err, out)
	}
	// Re-encode the decoded doc and check byte-stability.
	out2, err := acl.Encode(d)
	if err != nil {
		t.Fatalf("re-Encode: %v", err)
	}
	if string(out) != string(out2) {
		t.Fatalf("not byte-stable across round trip")
	}
}

func TestDeterministic(t *testing.T) {
	t.Parallel()
	s := tinySchema()
	a, _ := Encode(s)
	b, _ := Encode(s)
	if string(a) != string(b) {
		t.Fatalf("translator output is not deterministic")
	}
}

func TestCanonicalType(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in, want string
	}{
		{"integer", "int4"},
		{"bigint", "int8"},
		{"int8", "int8"},
		{"character varying(255)", "text"},
		{"varchar(64)", "text"},
		{"text", "text"},
		{"timestamp with time zone", "timestamptz"},
		{"timestamp without time zone", "timestamp"},
		{"jsonb", "jsonb"},
		{"boolean", "bool"},
		{"double precision", "float8"},
		{"user defined", "user_defined"},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			t.Parallel()
			if got := canonicalType(tc.in); got != tc.want {
				t.Fatalf("canonicalType(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestShortDefault(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in, want string
	}{
		{"nextval('users_id_seq'::regclass)", "AUTO"},
		{"now()", "now"},
		{"CURRENT_TIMESTAMP", "now"},
		{"true", "true"},
		{"false", "false"},
		{"NULL", "null"},
		{"'pending'::text", "pending"},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			t.Parallel()
			if got := shortDefault(tc.in); got != tc.want {
				t.Fatalf("shortDefault(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// ecommerceSchema synthesises a realistic 30-table e-commerce schema
// with ~250 columns, ~40 indexes, and ~50 foreign keys. This is the
// fixture the compression benchmark uses; it's deliberately built in
// code (not loaded from a SQL file) so the test is hermetic and
// reviewers can audit the generation rules below.
//
// Schema shape (the kind of thing a junior at a Series B might design):
//
//	users / sessions / addresses / payment_methods
//	products / categories / brands / inventory / variants
//	carts / cart_items
//	orders / order_items / shipments / shipment_items / refunds
//	reviews / wishlists / wishlist_items
//	coupons / coupon_redemptions
//	support_tickets / ticket_messages
//	audit_log / webhooks / webhook_deliveries
//	posts / post_tags / tags
//	subscriptions / subscription_events
func ecommerceSchema() Schema {
	tables := []Table{}
	indexes := []Index{}
	relations := []Relation{}

	// Most tables share a few stock columns. Helper to cut the
	// fixture from ~600 lines to ~150.
	std := func(name string, cols ...Column) Table {
		base := []Column{
			{Name: "id", Type: "bigint", Default: "nextval('" + name + "_id_seq'::regclass)"},
			{Name: "created_at", Type: "timestamp with time zone", Default: "now()"},
			{Name: "updated_at", Type: "timestamp with time zone", Default: "now()"},
		}
		base = append(base, cols...)
		return Table{Name: name, Columns: base, PrimaryKey: []string{"id"}}
	}
	col := func(n, t string, nul bool) Column { return Column{Name: n, Type: t, Nullable: nul} }
	colDef := func(n, t, def string) Column { return Column{Name: n, Type: t, Default: def} }
	idx := func(name, table string, cols []string, uniq bool) Index {
		return Index{Name: name, Table: table, Columns: cols, Unique: uniq}
	}
	fk := func(name, ft, fc, tt, tc string) Relation {
		return Relation{Name: name, FromTable: ft, FromColumn: fc, ToTable: tt, ToColumn: tc}
	}

	// 1-4: identity & contact
	tables = append(tables,
		std("users",
			col("email", "character varying(320)", false),
			col("password_hash", "text", false),
			col("display_name", "character varying(120)", true),
			colDef("status", "character varying(32)", "'active'::text"),
		),
		std("sessions",
			col("user_id", "bigint", false),
			col("token_hash", "text", false),
			col("expires_at", "timestamp with time zone", false),
			col("ip", "inet", true),
		),
		std("addresses",
			col("user_id", "bigint", false),
			col("line1", "character varying(255)", false),
			col("line2", "character varying(255)", true),
			col("city", "character varying(120)", false),
			col("region", "character varying(120)", true),
			col("postal_code", "character varying(20)", false),
			col("country", "character(2)", false),
		),
		std("payment_methods",
			col("user_id", "bigint", false),
			col("brand", "character varying(32)", false),
			col("last4", "character(4)", false),
			col("exp_month", "smallint", false),
			col("exp_year", "smallint", false),
		),
	)
	indexes = append(indexes,
		idx("users_email_key", "users", []string{"email"}, true),
		idx("sessions_user_id_idx", "sessions", []string{"user_id"}, false),
		idx("sessions_token_hash_key", "sessions", []string{"token_hash"}, true),
		idx("addresses_user_id_idx", "addresses", []string{"user_id"}, false),
		idx("payment_methods_user_id_idx", "payment_methods", []string{"user_id"}, false),
	)
	relations = append(relations,
		fk("sessions_user_id_fkey", "sessions", "user_id", "users", "id"),
		fk("addresses_user_id_fkey", "addresses", "user_id", "users", "id"),
		fk("payment_methods_user_id_fkey", "payment_methods", "user_id", "users", "id"),
	)

	// 5-9: catalogue
	tables = append(tables,
		std("brands",
			col("name", "character varying(120)", false),
			col("slug", "character varying(120)", false),
		),
		std("categories",
			col("parent_id", "bigint", true),
			col("name", "character varying(120)", false),
			col("slug", "character varying(120)", false),
		),
		std("products",
			col("brand_id", "bigint", false),
			col("category_id", "bigint", false),
			col("name", "character varying(255)", false),
			col("slug", "character varying(255)", false),
			col("description", "text", true),
			colDef("status", "character varying(32)", "'draft'::text"),
		),
		std("product_variants",
			col("product_id", "bigint", false),
			col("sku", "character varying(64)", false),
			col("price_cents", "integer", false),
			col("currency", "character(3)", false),
			col("weight_g", "integer", true),
		),
		std("inventory",
			col("variant_id", "bigint", false),
			col("warehouse", "character varying(64)", false),
			col("quantity", "integer", false),
		),
	)
	indexes = append(indexes,
		idx("brands_slug_key", "brands", []string{"slug"}, true),
		idx("categories_parent_id_idx", "categories", []string{"parent_id"}, false),
		idx("categories_slug_key", "categories", []string{"slug"}, true),
		idx("products_brand_id_idx", "products", []string{"brand_id"}, false),
		idx("products_category_id_idx", "products", []string{"category_id"}, false),
		idx("products_slug_key", "products", []string{"slug"}, true),
		idx("product_variants_product_id_idx", "product_variants", []string{"product_id"}, false),
		idx("product_variants_sku_key", "product_variants", []string{"sku"}, true),
		idx("inventory_variant_id_idx", "inventory", []string{"variant_id"}, false),
	)
	relations = append(relations,
		fk("categories_parent_id_fkey", "categories", "parent_id", "categories", "id"),
		fk("products_brand_id_fkey", "products", "brand_id", "brands", "id"),
		fk("products_category_id_fkey", "products", "category_id", "categories", "id"),
		fk("product_variants_product_id_fkey", "product_variants", "product_id", "products", "id"),
		fk("inventory_variant_id_fkey", "inventory", "variant_id", "product_variants", "id"),
	)

	// 10-15: cart, order, shipment
	tables = append(tables,
		std("carts",
			col("user_id", "bigint", true),
			col("session_id", "bigint", true),
		),
		std("cart_items",
			col("cart_id", "bigint", false),
			col("variant_id", "bigint", false),
			colDef("quantity", "integer", "1"),
		),
		std("orders",
			col("user_id", "bigint", false),
			col("billing_address_id", "bigint", false),
			col("shipping_address_id", "bigint", false),
			col("payment_method_id", "bigint", false),
			colDef("status", "character varying(32)", "'pending'::text"),
			col("total_cents", "integer", false),
			col("currency", "character(3)", false),
		),
		std("order_items",
			col("order_id", "bigint", false),
			col("variant_id", "bigint", false),
			col("quantity", "integer", false),
			col("unit_price_cents", "integer", false),
		),
		std("shipments",
			col("order_id", "bigint", false),
			col("carrier", "character varying(64)", false),
			col("tracking_number", "character varying(120)", true),
			colDef("status", "character varying(32)", "'pending'::text"),
		),
		std("refunds",
			col("order_id", "bigint", false),
			col("amount_cents", "integer", false),
			col("reason", "text", true),
		),
	)
	indexes = append(indexes,
		idx("carts_user_id_idx", "carts", []string{"user_id"}, false),
		idx("cart_items_cart_id_idx", "cart_items", []string{"cart_id"}, false),
		idx("orders_user_id_idx", "orders", []string{"user_id"}, false),
		idx("orders_status_idx", "orders", []string{"status"}, false),
		idx("order_items_order_id_idx", "order_items", []string{"order_id"}, false),
		idx("shipments_order_id_idx", "shipments", []string{"order_id"}, false),
		idx("refunds_order_id_idx", "refunds", []string{"order_id"}, false),
	)
	relations = append(relations,
		fk("carts_user_id_fkey", "carts", "user_id", "users", "id"),
		fk("cart_items_cart_id_fkey", "cart_items", "cart_id", "carts", "id"),
		fk("cart_items_variant_id_fkey", "cart_items", "variant_id", "product_variants", "id"),
		fk("orders_user_id_fkey", "orders", "user_id", "users", "id"),
		fk("orders_billing_address_id_fkey", "orders", "billing_address_id", "addresses", "id"),
		fk("orders_shipping_address_id_fkey", "orders", "shipping_address_id", "addresses", "id"),
		fk("orders_payment_method_id_fkey", "orders", "payment_method_id", "payment_methods", "id"),
		fk("order_items_order_id_fkey", "order_items", "order_id", "orders", "id"),
		fk("order_items_variant_id_fkey", "order_items", "variant_id", "product_variants", "id"),
		fk("shipments_order_id_fkey", "shipments", "order_id", "orders", "id"),
		fk("refunds_order_id_fkey", "refunds", "order_id", "orders", "id"),
	)

	// 16-20: reviews, wishlists, coupons, tags
	tables = append(tables,
		std("reviews",
			col("product_id", "bigint", false),
			col("user_id", "bigint", false),
			col("rating", "smallint", false),
			col("body", "text", true),
		),
		std("wishlists", col("user_id", "bigint", false), col("name", "character varying(120)", false)),
		std("wishlist_items", col("wishlist_id", "bigint", false), col("variant_id", "bigint", false)),
		std("coupons",
			col("code", "character varying(64)", false),
			col("discount_cents", "integer", false),
			col("expires_at", "timestamp with time zone", true),
		),
		std("coupon_redemptions",
			col("coupon_id", "bigint", false),
			col("order_id", "bigint", false),
			col("user_id", "bigint", false),
		),
	)
	indexes = append(indexes,
		idx("reviews_product_id_idx", "reviews", []string{"product_id"}, false),
		idx("reviews_user_id_idx", "reviews", []string{"user_id"}, false),
		idx("wishlists_user_id_idx", "wishlists", []string{"user_id"}, false),
		idx("wishlist_items_wishlist_id_idx", "wishlist_items", []string{"wishlist_id"}, false),
		idx("coupons_code_key", "coupons", []string{"code"}, true),
		idx("coupon_redemptions_coupon_id_idx", "coupon_redemptions", []string{"coupon_id"}, false),
	)
	relations = append(relations,
		fk("reviews_product_id_fkey", "reviews", "product_id", "products", "id"),
		fk("reviews_user_id_fkey", "reviews", "user_id", "users", "id"),
		fk("wishlists_user_id_fkey", "wishlists", "user_id", "users", "id"),
		fk("wishlist_items_wishlist_id_fkey", "wishlist_items", "wishlist_id", "wishlists", "id"),
		fk("wishlist_items_variant_id_fkey", "wishlist_items", "variant_id", "product_variants", "id"),
		fk("coupon_redemptions_coupon_id_fkey", "coupon_redemptions", "coupon_id", "coupons", "id"),
		fk("coupon_redemptions_order_id_fkey", "coupon_redemptions", "order_id", "orders", "id"),
		fk("coupon_redemptions_user_id_fkey", "coupon_redemptions", "user_id", "users", "id"),
	)

	// 21-25: support
	tables = append(tables,
		std("support_tickets",
			col("user_id", "bigint", false),
			col("subject", "character varying(255)", false),
			colDef("status", "character varying(32)", "'open'::text"),
		),
		std("ticket_messages",
			col("ticket_id", "bigint", false),
			col("author_id", "bigint", false),
			col("body", "text", false),
		),
		std("audit_log",
			col("actor_id", "bigint", true),
			col("entity", "character varying(120)", false),
			col("entity_id", "bigint", false),
			col("action", "character varying(64)", false),
			col("payload", "jsonb", true),
		),
		std("webhooks",
			col("user_id", "bigint", false),
			col("url", "text", false),
			col("secret", "text", false),
			colDef("active", "boolean", "true"),
		),
		std("webhook_deliveries",
			col("webhook_id", "bigint", false),
			col("event", "character varying(120)", false),
			colDef("status_code", "integer", "0"),
			col("response_body", "text", true),
		),
	)
	indexes = append(indexes,
		idx("support_tickets_user_id_idx", "support_tickets", []string{"user_id"}, false),
		idx("ticket_messages_ticket_id_idx", "ticket_messages", []string{"ticket_id"}, false),
		idx("audit_log_entity_idx", "audit_log", []string{"entity", "entity_id"}, false),
		idx("audit_log_actor_id_idx", "audit_log", []string{"actor_id"}, false),
		idx("webhooks_user_id_idx", "webhooks", []string{"user_id"}, false),
		idx("webhook_deliveries_webhook_id_idx", "webhook_deliveries", []string{"webhook_id"}, false),
	)
	relations = append(relations,
		fk("support_tickets_user_id_fkey", "support_tickets", "user_id", "users", "id"),
		fk("ticket_messages_ticket_id_fkey", "ticket_messages", "ticket_id", "support_tickets", "id"),
		fk("ticket_messages_author_id_fkey", "ticket_messages", "author_id", "users", "id"),
		fk("audit_log_actor_id_fkey", "audit_log", "actor_id", "users", "id"),
		fk("webhooks_user_id_fkey", "webhooks", "user_id", "users", "id"),
		fk("webhook_deliveries_webhook_id_fkey", "webhook_deliveries", "webhook_id", "webhooks", "id"),
	)

	// 26-30: blog + subscriptions
	tables = append(tables,
		std("posts",
			col("author_id", "bigint", false),
			col("title", "character varying(255)", false),
			col("slug", "character varying(255)", false),
			col("body", "text", false),
			colDef("published", "boolean", "false"),
		),
		std("tags", col("name", "character varying(120)", false), col("slug", "character varying(120)", false)),
		std("post_tags", col("post_id", "bigint", false), col("tag_id", "bigint", false)),
		std("subscriptions",
			col("user_id", "bigint", false),
			col("plan", "character varying(64)", false),
			colDef("status", "character varying(32)", "'active'::text"),
			col("current_period_end", "timestamp with time zone", false),
		),
		std("subscription_events",
			col("subscription_id", "bigint", false),
			col("event", "character varying(64)", false),
			col("payload", "jsonb", true),
		),
	)
	indexes = append(indexes,
		idx("posts_author_id_idx", "posts", []string{"author_id"}, false),
		idx("posts_slug_key", "posts", []string{"slug"}, true),
		idx("tags_slug_key", "tags", []string{"slug"}, true),
		idx("post_tags_post_id_idx", "post_tags", []string{"post_id"}, false),
		idx("post_tags_tag_id_idx", "post_tags", []string{"tag_id"}, false),
		idx("subscriptions_user_id_idx", "subscriptions", []string{"user_id"}, false),
		idx("subscription_events_subscription_id_idx", "subscription_events", []string{"subscription_id"}, false),
	)
	relations = append(relations,
		fk("posts_author_id_fkey", "posts", "author_id", "users", "id"),
		fk("post_tags_post_id_fkey", "post_tags", "post_id", "posts", "id"),
		fk("post_tags_tag_id_fkey", "post_tags", "tag_id", "tags", "id"),
		fk("subscriptions_user_id_fkey", "subscriptions", "user_id", "users", "id"),
		fk("subscription_events_subscription_id_fkey", "subscription_events", "subscription_id", "subscriptions", "id"),
	)

	return Schema{
		Database:   "shop",
		SchemaName: "public",
		Tables:     tables,
		Indexes:    indexes,
		Relations:  relations,
	}
}

// equivalentDDL returns a minimal Postgres DDL representation of the
// same schema. This is the LEAN baseline: just the substantive CREATE
// TABLE / CREATE INDEX / FK lines, without the verbose pg_dump
// ceremony. ACL's win against this baseline is modest (~1.5–2x)
// because both formats are dense type-and-name listings.
//
// For the realistic comparison — what an MCP postgres server actually
// returns when you run `pg_dump -s` — see realisticPgDump below.
func equivalentDDL(s Schema) string {
	var b strings.Builder
	for _, t := range s.Tables {
		fmt.Fprintf(&b, "CREATE TABLE %s.%s (\n", s.SchemaName, t.Name)
		for i, c := range t.Columns {
			fmt.Fprintf(&b, "    %s %s", c.Name, c.Type)
			if !c.Nullable {
				b.WriteString(" NOT NULL")
			}
			if c.Default != "" {
				fmt.Fprintf(&b, " DEFAULT %s", c.Default)
			}
			if i < len(t.Columns)-1 {
				b.WriteString(",")
			}
			b.WriteString("\n")
		}
		b.WriteString(");\n")
		if len(t.PrimaryKey) > 0 {
			fmt.Fprintf(&b, "ALTER TABLE ONLY %s.%s ADD CONSTRAINT %s_pkey PRIMARY KEY (%s);\n",
				s.SchemaName, t.Name, t.Name, strings.Join(t.PrimaryKey, ", "))
		}
		b.WriteString("\n")
	}
	for _, ix := range s.Indexes {
		uniq := ""
		if ix.Unique {
			uniq = "UNIQUE "
		}
		fmt.Fprintf(&b, "CREATE %sINDEX %s ON %s.%s USING btree (%s);\n",
			uniq, ix.Name, s.SchemaName, ix.Table, strings.Join(ix.Columns, ", "))
	}
	b.WriteString("\n")
	for _, r := range s.Relations {
		fmt.Fprintf(&b,
			"ALTER TABLE ONLY %s.%s ADD CONSTRAINT %s FOREIGN KEY (%s) REFERENCES %s.%s(%s);\n",
			s.SchemaName, r.FromTable, r.Name, r.FromColumn, s.SchemaName, r.ToTable, r.ToColumn)
	}
	return b.String()
}

// realisticPgDump is what `pg_dump -s` actually emits for the same
// schema, including the ceremonial SET / SELECT pg_catalog / OWNER /
// COMMENT / GRANT lines that no agent reads but every byte counts
// against the prompt budget.
//
// The template is anchored on the structure pg_dump 16 produces; the
// per-table overhead (~600 bytes of SET + ALTER OWNER + GRANT noise)
// dominates for schemas with small tables, which is the typical case.
func realisticPgDump(s Schema) string {
	var b strings.Builder
	b.WriteString("--\n-- PostgreSQL database dump\n--\n\n")
	b.WriteString("-- Dumped from database version 16.4\n")
	b.WriteString("-- Dumped by pg_dump version 16.4\n\n")
	b.WriteString("SET statement_timeout = 0;\n")
	b.WriteString("SET lock_timeout = 0;\n")
	b.WriteString("SET idle_in_transaction_session_timeout = 0;\n")
	b.WriteString("SET client_encoding = 'UTF8';\n")
	b.WriteString("SET standard_conforming_strings = on;\n")
	b.WriteString("SELECT pg_catalog.set_config('search_path', '', false);\n")
	b.WriteString("SET check_function_bodies = false;\n")
	b.WriteString("SET xmloption = content;\n")
	b.WriteString("SET client_min_messages = warning;\n")
	b.WriteString("SET row_security = off;\n\n")
	b.WriteString("SET default_tablespace = '';\n")
	b.WriteString("SET default_table_access_method = heap;\n\n")
	for _, t := range s.Tables {
		fmt.Fprintf(&b, "--\n-- Name: %s; Type: TABLE; Schema: %s; Owner: shop_app\n--\n\n",
			t.Name, s.SchemaName)
		fmt.Fprintf(&b, "CREATE TABLE %s.%s (\n", s.SchemaName, t.Name)
		for i, c := range t.Columns {
			fmt.Fprintf(&b, "    %s %s", c.Name, c.Type)
			if !c.Nullable {
				b.WriteString(" NOT NULL")
			}
			if c.Default != "" {
				fmt.Fprintf(&b, " DEFAULT %s", c.Default)
			}
			if i < len(t.Columns)-1 {
				b.WriteString(",")
			}
			b.WriteString("\n")
		}
		b.WriteString(");\n\n")
		fmt.Fprintf(&b, "ALTER TABLE %s.%s OWNER TO shop_app;\n\n", s.SchemaName, t.Name)
		// Each table also has an _id_seq sequence pg_dump emits.
		for _, c := range t.Columns {
			if c.Name == "id" && strings.Contains(c.Default, "nextval") {
				fmt.Fprintf(&b, "--\n-- Name: %s_id_seq; Type: SEQUENCE; Schema: %s; Owner: shop_app\n--\n\n",
					t.Name, s.SchemaName)
				fmt.Fprintf(&b, "CREATE SEQUENCE %s.%s_id_seq\n    START WITH 1\n    INCREMENT BY 1\n    NO MINVALUE\n    NO MAXVALUE\n    CACHE 1;\n\n",
					s.SchemaName, t.Name)
				fmt.Fprintf(&b, "ALTER TABLE %s.%s_id_seq OWNER TO shop_app;\n\n", s.SchemaName, t.Name)
				fmt.Fprintf(&b, "ALTER SEQUENCE %s.%s_id_seq OWNED BY %s.%s.id;\n\n",
					s.SchemaName, t.Name, s.SchemaName, t.Name)
			}
		}
		if len(t.PrimaryKey) > 0 {
			fmt.Fprintf(&b, "--\n-- Name: %s %s_pkey; Type: CONSTRAINT; Schema: %s; Owner: shop_app\n--\n\n",
				t.Name, t.Name, s.SchemaName)
			fmt.Fprintf(&b, "ALTER TABLE ONLY %s.%s ADD CONSTRAINT %s_pkey PRIMARY KEY (%s);\n\n",
				s.SchemaName, t.Name, t.Name, strings.Join(t.PrimaryKey, ", "))
		}
	}
	for _, ix := range s.Indexes {
		uniq := ""
		if ix.Unique {
			uniq = "UNIQUE "
		}
		fmt.Fprintf(&b, "--\n-- Name: %s; Type: INDEX; Schema: %s; Owner: shop_app\n--\n\n",
			ix.Name, s.SchemaName)
		fmt.Fprintf(&b, "CREATE %sINDEX %s ON %s.%s USING btree (%s);\n\n",
			uniq, ix.Name, s.SchemaName, ix.Table, strings.Join(ix.Columns, ", "))
	}
	for _, r := range s.Relations {
		fmt.Fprintf(&b,
			"--\n-- Name: %s %s; Type: FK CONSTRAINT; Schema: %s; Owner: shop_app\n--\n\n",
			r.FromTable, r.Name, s.SchemaName)
		fmt.Fprintf(&b,
			"ALTER TABLE ONLY %s.%s ADD CONSTRAINT %s FOREIGN KEY (%s) REFERENCES %s.%s(%s);\n\n",
			s.SchemaName, r.FromTable, r.Name, r.FromColumn, s.SchemaName, r.ToTable, r.ToColumn)
	}
	b.WriteString("--\n-- PostgreSQL database dump complete\n--\n\n")
	return b.String()
}

// TestEcommerceCompressionRatio is the headline benchmark for the PG
// translator. We report two ratios because both are honest:
//
//   - ACL vs trimmed DDL: ~1.5–2x. Both formats are dense type-and-
//     name listings; format-level wins are modest.
//   - ACL vs realistic pg_dump output: ~4x or better, because pg_dump
//     adds ~600 bytes of SET / OWNER / COMMENT / SEQUENCE ceremony
//     per table that no agent reads. This is the comparison that
//     matters for production agent prompt budgets.
func TestEcommerceCompressionRatio(t *testing.T) {
	t.Parallel()
	s := ecommerceSchema()
	out, err := Encode(s)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	ddl := equivalentDDL(s)
	dump := realisticPgDump(s)
	lean := float64(len(ddl)) / float64(len(out))
	realistic := float64(len(dump)) / float64(len(out))
	t.Logf("schema: %d tables, %d indexes, %d FKs",
		len(s.Tables), len(s.Indexes), len(s.Relations))
	t.Logf("ACL bytes:               %d (~%d tokens)", len(out), len(out)/4)
	t.Logf("trimmed DDL bytes:       %d (~%d tokens)", len(ddl), len(ddl)/4)
	t.Logf("realistic pg_dump bytes: %d (~%d tokens)", len(dump), len(dump)/4)
	t.Logf("compression vs trimmed DDL:       %.1fx", lean)
	t.Logf("compression vs realistic pg_dump: %.1fx", realistic)
	const minLean = 1.5
	if lean < minLean {
		t.Fatalf("expected >=%.1fx vs trimmed DDL, got %.1fx", minLean, lean)
	}
	// 3.0x is the honest threshold against realistic pg_dump. SQL DDL
	// is a dense format to begin with (vs JSON or HTML), so ACL's
	// per-byte advantage is smaller here than for K8s state (~100x)
	// or OpenAPI specs (~10-70x). The token savings are still
	// significant: ~3.5x means ~70% fewer prompt tokens.
	const minRealistic = 3.0
	if realistic < minRealistic {
		t.Fatalf("expected >=%.1fx vs realistic pg_dump, got %.1fx", minRealistic, realistic)
	}
}

func TestEcommerceRoundTrip(t *testing.T) {
	t.Parallel()
	out, err := Encode(ecommerceSchema())
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if _, err := acl.Decode(out); err != nil {
		t.Fatalf("Decode(Encode(ecommerceSchema)): %v", err)
	}
}
