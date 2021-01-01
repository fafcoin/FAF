/*
  This file is part of fafash.

  fafash is free software: you can redistribute it and/or modify
  it under the terms of the GNU General Public License as published by
  the Free Software Foundation, either version 3 of the License, or
  (at your option) any later version.

  fafash is distributed in the hope that it will be useful,
  but WITHOUT ANY WARRANTY; without even the implied warranty of
  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
  GNU General Public License for more details.

  You should have received a copy of the GNU General Public License
  along with fafash.  If not, see <http://www.gnu.org/licenses/>.
*/

/** @file fafash.h
* @date 2015
*/
#pragma once

#include <stdint.h>
#include <stdbool.h>
#include <string.h>
#include <stddef.h>
#include "compiler.h"

#define fafASH_REVISION 23
#define fafASH_DATASET_BYTES_INIT 1073741824U // 2**30
#define fafASH_DATASET_BYTES_GROWTH 8388608U  // 2**23
#define fafASH_CACHE_BYTES_INIT 1073741824U // 2**24
#define fafASH_CACHE_BYTES_GROWTH 131072U  // 2**17
#define fafASH_EPOCH_LENGTH 30000U
#define fafASH_MIX_BYTES 128
#define fafASH_HASH_BYTES 64
#define fafASH_DATASET_PARENTS 256
#define fafASH_CACHE_ROUNDS 3
#define fafASH_ACCESSES 64
#define fafASH_DAG_MAGIC_NUM_SIZE 8
#define fafASH_DAG_MAGIC_NUM 0xFEE1DEADBADDCAFE

#ifdef __cplusplus
extern "C" {
#endif

/// Type of a seedhash/blockhash e.t.c.
typedef struct fafash_h256 { uint8_t b[32]; } fafash_h256_t;

// convenience macro to statically initialize an h256_t
// usage:
// fafash_h256_t a = fafash_h256_static_init(1, 2, 3, ... )
// have to provide all 32 values. If you don't provide all the rest
// will simply be unitialized (not guranteed to be 0)
#define fafash_h256_static_init(...)			\
	{ {__VA_ARGS__} }

struct fafash_light;
typedef struct fafash_light* fafash_light_t;
struct fafash_full;
typedef struct fafash_full* fafash_full_t;
typedef int(*fafash_callback_t)(unsigned);

typedef struct fafash_return_value {
	fafash_h256_t result;
	fafash_h256_t mix_hash;
	bool success;
} fafash_return_value_t;

/**
 * Allocate and initialize a new fafash_light handler
 *
 * @param block_number   The block number for which to create the handler
 * @return               Newly allocated fafash_light handler or NULL in case of
 *                       ERRNOMEM or invalid parameters used for @ref fafash_compute_cache_nodes()
 */
fafash_light_t fafash_light_new(uint64_t block_number);
/**
 * Frees a previously allocated fafash_light handler
 * @param light        The light handler to free
 */
void fafash_light_delete(fafash_light_t light);
/**
 * Calculate the light client data
 *
 * @param light          The light client handler
 * @param header_hash    The header hash to pack into the mix
 * @param nonce          The nonce to pack into the mix
 * @return               an object of fafash_return_value_t holding the return values
 */
fafash_return_value_t fafash_light_compute(
	fafash_light_t light,
	fafash_h256_t const header_hash,
	uint64_t nonce
);

/**
 * Allocate and initialize a new fafash_full handler
 *
 * @param light         The light handler containing the cache.
 * @param callback      A callback function with signature of @ref fafash_callback_t
 *                      It accepts an unsigned with which a progress of DAG calculation
 *                      can be displayed. If all goes well the callback should return 0.
 *                      If a non-zero value is returned then DAG generation will stop.
 *                      Be advised. A progress value of 100 means that DAG creation is
 *                      almost complete and that this function will soon return succesfully.
 *                      It does not mean that the function has already had a succesfull return.
 * @return              Newly allocated fafash_full handler or NULL in case of
 *                      ERRNOMEM or invalid parameters used for @ref fafash_compute_full_data()
 */
fafash_full_t fafash_full_new(fafash_light_t light, fafash_callback_t callback);

/**
 * Frees a previously allocated fafash_full handler
 * @param full    The light handler to free
 */
void fafash_full_delete(fafash_full_t full);
/**
 * Calculate the full client data
 *
 * @param full           The full client handler
 * @param header_hash    The header hash to pack into the mix
 * @param nonce          The nonce to pack into the mix
 * @return               An object of fafash_return_value to hold the return value
 */
fafash_return_value_t fafash_full_compute(
	fafash_full_t full,
	fafash_h256_t const header_hash,
	uint64_t nonce
);
/**
 * Get a pointer to the full DAG data
 */
void const* fafash_full_dag(fafash_full_t full);
/**
 * Get the size of the DAG data
 */
uint64_t fafash_full_dag_size(fafash_full_t full);

/**
 * Calculate the seedhash for a given block number
 */
fafash_h256_t fafash_get_seedhash(uint64_t block_number);

#ifdef __cplusplus
}
#endif
