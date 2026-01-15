PUT http://localhost:9200/perfumes
Content-Type: application/json

{
  "settings": {
    "number_of_shards": 1,
    "number_of_replicas": 0,
    "analysis": {
      "filter": {
        "ru_stop": { "type": "stop", "stopwords": "_russian_" },
        "en_stop": { "type": "stop", "stopwords": "_english_" },
        "ru_stemmer": { "type": "stemmer", "language": "russian" },
        "en_stemmer": { "type": "stemmer", "language": "english" }
      },
      "analyzer": {
        "mix_ru_en": {
          "type": "custom",
          "tokenizer": "standard",
          "filter": ["lowercase", "asciifolding", "ru_stop", "en_stop", "ru_stemmer", "en_stemmer"]
        }
      }
    }
  },
  "mappings": {
    "properties": {
      "id": { "type": "long" },
      "url": { "type": "keyword" },

      "name": {
        "type": "text",
        "analyzer": "mix_ru_en",
        "fields": { "keyword": { "type": "keyword" } }
      },

      "brand_en": { "type": "text", "analyzer": "mix_ru_en", "fields": { "keyword": { "type": "keyword" } } },

      "gender_en": { "type": "keyword" },
      "gender_ru": { "type": "keyword" },
      "year": { "type": "integer" },

      "rating_value": { "type": "float" },
      "rating_count": { "type": "integer" },

      "notes_en": { "type": "text", "analyzer": "mix_ru_en" },
      "notes_ru": { "type": "text", "analyzer": "mix_ru_en" },

      "accords_en": { "type": "text", "analyzer": "mix_ru_en" },
      "accords_ru": { "type": "text", "analyzer": "mix_ru_en" },

      "perfumers_en": { "type": "text", "analyzer": "mix_ru_en", "fields": { "keyword": { "type": "keyword" } } }
    }
  }
}