import random
import numpy as np
from collections import defaultdict

##### SETTINGS FOR THE GENERATOR #####

# Files to write to.
data_file = "../data/PrometheusDataSecond.csv"
results_file = "../data/PrometheusAnswersFirst.csv"

# Number of records (lines in csv file) to generate
num_records = 2000

# Number of values to add to a checkpoint set. This simulates how many values are recorded by an instrument.
num_values = 8

# Range of values to generate.
value_lower_limit = -50
value_upper_limit = 50

# Histogram boundaries for buckets. Buckets in the Go SDK include "lower" buckets -- For example, if values are from 0 to 1
# and there is a boundary at 0.5, the buckets would be (-int, 0.5) and (-inf, +inf) instead of (-inf, 0.5), [0.5, +inf]
histogram_boundaries = [-25, 0, 25]

# Quantiles for distributions.
quantiles = [0.25, 0.5, 0.75]

# List of all aggregation types in the OTel Go SDK.
aggregations = ["hist", "dist", "sum", "mmsc", "lval"]

# A 2D dictionary of answers. Rows represent aggregation types and columns hold properties (name, description, label).
# Each individual dictionary element is a list of 11 elements, which represent:
# 0: Final Balue (sum / last value)
# 1. Min
# 2. Max
# 3. Count
# 4. (-inf, -25) bucket
# 5. (-inf, 0) bucket 
# 6. (-inf, 25) bucket
# 7. (-inf, +inf) bucket
# 8. 0.25 quantile
# 9. 0.5 quantile
# 10. 0.75 quantile.
answers = defaultdict(lambda: defaultdict(lambda: [0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0]))

##### GENERATING DATA FILE #####

# Open the data file to write to.
f = open(data_file, "w")

# Write `num_records` records to the file.
for i in range(num_records):
    # Randomly select an aggregation type.
    agg_type = random.choice(aggregations)

    # Create unique strings for the name, description, and label.
    name = f"p2name{i}_{agg_type}"
    description = f"description{i}"
    label = f"{{key{i}:value{i}}}"

    # Create a properties string that identifies the record with the name, description, and label.
    agg_properties = f"{name},{description},{label}"

    # Generate a list of `num_values` random values that will be used to update the CheckpointSet.
    values = random.sample(range(value_lower_limit, value_upper_limit), num_values)

    # Write different types of records depending on the aggregation type.
    record = f"{agg_type}|{str(values).replace(' ', '')}|{agg_properties}"
    if agg_type == "sum":
        # Final value (sum).
        answers["sum"][agg_properties][0] = sum(values)

    elif agg_type == "lval":
        # Final value (last value).
        answers["lval"][agg_properties][0] = values[len(values) - 1]

    elif agg_type == "mmsc":
        # Final value (sum), min, max, and count.
        answers["mmsc"][agg_properties][0] = sum(values)
        answers["mmsc"][agg_properties][1] = min(values)
        answers["mmsc"][agg_properties][2] = max(values)
        answers["mmsc"][agg_properties][3] = num_values

    # Distribution aggregations are MinMaxSumCount aggregations with quantiles.
    elif agg_type == "dist":
        # Final value (sum), min, max, and count.
        answers["dist"][agg_properties][0] = sum(values)
        answers["dist"][agg_properties][1] = min(values)
        answers["dist"][agg_properties][2] = max(values)
        answers["dist"][agg_properties][3] = num_values

        # Quantiles are calculated using numpy.
        values_numpy = np.array(values)
        answers["dist"][agg_properties][8] = int(np.percentile(values_numpy, quantiles[0]))
        answers["dist"][agg_properties][9] = int(np.percentile(values_numpy, quantiles[1]))
        answers["dist"][agg_properties][10] = int(np.percentile(values_numpy, quantiles[2]))
        
    elif agg_type == "hist":
        # Final value (sum).
        answers["hist"][agg_properties][0] = sum(values)

        # Count.
        answers["hist"][agg_properties][3] = num_values

        # (-inf, -25) bucket.
        answers["hist"][agg_properties][4] = len([i for i in values if i < -25])

        # (-inf, 0) bucket
        answers["hist"][agg_properties][5] = len([i for i in values if i < 0])

        # (-inf, 25) bucket
        answers["hist"][agg_properties][6] = len([i for i in values if i < 25])

        # (-inf, +inf) bucket
        answers["hist"][agg_properties][7] = num_values

    # Write the record to the file.
    f.write(record + "\n")
    
    
f.close()

##### GENERATING ANSWER FILE #####
f = open(results_file, 'w+')

# Iterate through every record in the answer dictionary. Note that order is not constant in a dictionary so the csv
# file may not be in order index wise.
for agg_type in answers:
    for agg_properties in answers[agg_type]:
        value = answers[agg_type][agg_properties][0]
        min = answers[agg_type][agg_properties][1]
        max = answers[agg_type][agg_properties][2]
        count = answers[agg_type][agg_properties][3]
        bucket_0 = answers[agg_type][agg_properties][4]
        bucket_1 = answers[agg_type][agg_properties][5]
        bucket_2 = answers[agg_type][agg_properties][6]
        bucket_3 = answers[agg_type][agg_properties][7]
        quantile_0 = answers[agg_type][agg_properties][8]
        quantile_1 = answers[agg_type][agg_properties][9]
        quantile_2 = answers[agg_type][agg_properties][10]

        # Prepare a record (row in csv file) that will be written to the csv file.
        record = ""

        # Create records based on what the aggregation type.
        if agg_type == "sum":
            record = f"{agg_properties}|{agg_type}|{value}"
        elif agg_type == "lval":
            record = f"{agg_properties}|{agg_type}|{value}"
        elif agg_type == "mmsc":
            record = f"{agg_properties}|{agg_type}|{value}|{min}|{max}|{count}"
        elif agg_type == "dist":
            record = f"{agg_properties}|{agg_type}|{value}|{min}|{max}|{count}|{{{quantile_0},{quantile_1},{quantile_2}}}"
        elif agg_type == "hist":
            record = f"{agg_properties}|{agg_type}|{value}|{count}|{{{bucket_0},{bucket_1},{bucket_2},{bucket_3}}}"

        # Write the full record to the csv file. 
        f.write(record + "\n")

# Save the records so it can be sorted later.
f.seek(0)
data = f.readlines()
    
# Close the file after writing.
f.close()

# Define custom key function so records are sorted based on its index.
def record_index(record):
    # Record always starts with "p1name<index>_". The index is retrieved using substrings.
    underscore_index = record.index('_')
    record_name = record[:underscore_index]
    return int(record_name[6:])

# Sort the data and then write it back to the file.
data = sorted(data, key=record_index)
f = open(results_file, 'w')
for record in data:
    f.write(record)

# Close the file after writing.
f.close()