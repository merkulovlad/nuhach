import pandas as pd
import numpy as np
import torch
from transformers import MarianMTModel, MarianTokenizer
from tqdm.auto import tqdm
from transliterate import translit


def clean_from_unknown(df: pd.DataFrame, column_name: str) -> pd.DataFrame:
    df = df.copy()
    """Removes rows with 'unknown' values in the given column by setting them to NaN."""
    df[column_name] = df[column_name].apply(
        lambda x: np.nan
        if pd.isna(x) or str(x).strip().lower() == "unknown"
        else x
    )
    return df


# translation model settings
_MODEL_NAME = "Helsinki-NLP/opus-mt-en-ru"
_tokenizer = None
_model = None


def _ensure_translation_model():
    global _tokenizer, _model
    if _tokenizer is None or _model is None:
        _tokenizer = MarianTokenizer.from_pretrained(_MODEL_NAME)
        _model = MarianMTModel.from_pretrained(_MODEL_NAME)


def translate_dataset_to_ru(df: pd.DataFrame, batch_size: int = 16) -> pd.DataFrame:
    """
    Translates selected perfume columns from EN to RU using a MarianMT model.
    Note: the `Perfume` column is intentionally NOT translated by the MT model
    (we avoid semantic translation of perfume names). Perfume may be
    transliterated later separately.
    NaN stays NaN. Returns new DataFrame.
    """

    cols_to_translate = [
        # Do NOT include "Perfume" here — we skip semantic translation of names
        "Brand", "Country",
        "Top", "Middle", "Base",
        "mainaccord1", "mainaccord2", "mainaccord3",
        "mainaccord4", "mainaccord5",
    ]

    df = df.copy()
    _ensure_translation_model()

    for col in cols_to_translate:
        if col not in df.columns:
            continue

        series = df[col]
        mask = series.notna()
        unique_texts = series[mask].unique()

        translations = {}

        total_batches = (len(unique_texts) + batch_size - 1) // batch_size

        for i in tqdm(range(0, len(unique_texts), batch_size),
                      desc=f"Translating {col}",
                      total=total_batches):
            batch = unique_texts[i:i+batch_size].tolist()
            encoded = _tokenizer(batch, return_tensors="pt", padding=True, truncation=True)

            with torch.no_grad():
                generated = _model.generate(**encoded)

            decoded = _tokenizer.batch_decode(generated, skip_special_tokens=True)
            translations.update(dict(zip(batch, decoded)))

        df[col] = series.map(translations)

    return df


def translate_gender_to_ru(df: pd.DataFrame) -> pd.DataFrame:
    """
    Translates gender column from EN to RU.
    'men' -> 'мужской'
    'women' -> 'женский'
    'unisex' -> 'унисекс'
    NaN stays NaN. Returns new DataFrame.
    """

    df = df.copy()
    gender_map = {
        "men": "мужской",
        "women": "женский",
        "unisex": "унисекс",
    }
    if "Gender" in df.columns:
        df["Gender_ru"] = df["Gender"].map(gender_map)
    else:
        df["Gender_ru"] = np.nan
    return df


def perfume_to_ru(x: str) -> str:
    """Transliterate perfume name to Cyrillic; do not semantically translate."""
    if pd.isna(x):
        return x

    x = str(x).strip()
    if not x:
        return x

    try:
        return translit(x, 'ru')
    except Exception:
        return x


def main():
    # Read raw dataset
    df = pd.read_csv(
        "../data/raw/fra_cleaned.csv",
        encoding="latin-1",
        sep=";",
    )

    # Clean and translate (excluding `Perfume` semantic translation)
    df = clean_from_unknown(df, "Perfumer1")
    df = translate_dataset_to_ru(df)
    df = translate_gender_to_ru(df)
    df.to_csv("../data/processed/dataset_translated.csv", index=False, encoding="utf-8")

    # Merge English original and translated data to produce final dataset
    df_en = pd.read_csv(
        "../data/raw/fra_cleaned.csv",
        encoding="latin-1",
        sep=";",
    )

    df_trans = pd.read_csv("../data/processed/dataset_translated.csv")
    df_trans = df_trans.drop(columns=["Perfume"], errors="ignore")

    df_final = df_en.merge(
        df_trans,
        on="url",
        how="inner",
        validate="one_to_one",
    )

    # Do not create or write a `Perfume_ru` column — keep original `Perfume` unchanged
    df_final.to_csv("../data/processed/dataset_final.csv", index=False, encoding="utf-8")


if __name__ == '__main__':
    main()
