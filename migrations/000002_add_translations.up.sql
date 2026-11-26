CREATE TABLE IF NOT EXISTS menu_item_translations (
    item_id INTEGER NOT NULL REFERENCES menu_items(id) ON DELETE CASCADE,
    lang VARCHAR(5) NOT NULL,
    name VARCHAR(100) NOT NULL,
    PRIMARY KEY (item_id, lang)
);

-- পুরনো নাম গুলো bn translation হিসেবে copy
INSERT INTO menu_item_translations (item_id, lang, name)
SELECT id, 'bn', name FROM menu_items WHERE name IS NOT NULL
ON CONFLICT DO NOTHING;

-- ponytail: পুরনো name column রেখে দিলাম (data হারাবে না), পরে চাইলে আলাদা migration এ drop করো
ALTER TABLE menu_items ALTER COLUMN name DROP NOT NULL;
