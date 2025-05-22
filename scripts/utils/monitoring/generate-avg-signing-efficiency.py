import pandas as pd
import sys
import os

'''
This script calculates the average signing efficiency of validators from a CSV file (CVMS data).
The CSV file should contain a 'Time' column and a 'moniker' column.
The script will calculate the average value for each validator, compute uptime percentage,
and save the results to a new CSV file.

Usage:
    python generate-avg-signing-efficiency.py <input_csv_path> [output_csv_path]
'''

def print_usage():
    print("Usage: python generate-avg-signing-efficiency.py <input_csv_path> [output_csv_path]")
    print("Example: python generate-avg-signing-efficiency.py exported_data.csv results.csv")

def main():
    if len(sys.argv) < 2 or len(sys.argv) > 3:
        print("Error: Invalid number of arguments.")
        print_usage()
        sys.exit(1)

    input_path = sys.argv[1]
    output_path = sys.argv[2] if len(sys.argv) == 3 else "moniker_averages_with_uptime.csv"

    if not os.path.isfile(input_path):
        print(f"Error: File not found: {input_path}")
        sys.exit(1)

    # Load the CSV file
    df = pd.read_csv(input_path)

    # Drop the 'Time' column and reshape the dataframe to long format
    if "Time" not in df.columns:
        print("Error: CSV does not contain a 'Time' column.")
        sys.exit(1)

    df_long = df.drop(columns=["Time"]).melt(var_name="moniker", value_name="value")

    # Convert values to numeric
    df_long["value"] = pd.to_numeric(df_long["value"], errors="coerce")

    # Trim whitespace from moniker names
    df_long["moniker"] = df_long["moniker"].str.strip()

    # Group by moniker and calculate average
    df_avg = df_long.groupby("moniker", as_index=False)["value"].mean()

    # Sort alphabetically, ignoring case
    df_avg_sorted = df_avg.sort_values(by="moniker", key=lambda col: col.str.lower())

    # Compute uptime percentage
    total_entries = df_long["value"].groupby(df_long["moniker"]).size()
    non_null_entries = df_long["value"].groupby(df_long["moniker"]).apply(lambda x: x.notnull().sum())
    uptime_percentage = (non_null_entries / total_entries * 100).reset_index(name="uptime_percentage")

    # Merge both
    df_final = pd.merge(df_avg_sorted, uptime_percentage, on="moniker")

    # Save to CSV
    df_final.to_csv(output_path, index=False)
    print(f"Output saved to: {output_path}")

if __name__ == "__main__":
    main()
