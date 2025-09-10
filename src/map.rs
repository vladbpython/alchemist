use dashmap::DashMap;
use std::{
    os::raw::c_int,
    slice,
};

#[repr(C)]
pub struct AlchemistMap {
    inner: DashMap<usize, usize>,
}

#[no_mangle]
pub extern "C" fn alchemist_map_new() -> *mut AlchemistMap {
    Box::into_raw(Box::new(AlchemistMap {
        inner: DashMap::new(),
    }))
}

#[no_mangle]
pub extern "C" fn alchemist_map_destroy(obj: *mut AlchemistMap) {
    if !obj.is_null() {
        unsafe { drop(Box::from_raw(obj)) }
    }
}

/// Вставка: ключ и значение как usize
#[no_mangle]
pub extern "C" fn alchemist_map_set(obj: *mut AlchemistMap, key: usize, val: usize) {
    if obj.is_null() || key == 0 || val == 0 {
        return
    }
    unsafe { (*obj).inner.insert(key, val); }
}

#[no_mangle]
pub extern "C" fn alchemist_map_get(obj: *mut AlchemistMap, key: usize) -> usize {
    if obj.is_null() || key == 0 {
        return 0
    }
    unsafe { (*obj).inner.get(&key).map(|v| *v).unwrap_or(0) }
}

#[no_mangle]
pub extern "C" fn alchemist_map_remove(obj: *mut AlchemistMap, key: usize) -> usize {
    if obj.is_null() || key == 0 {
        return 0
    }
    unsafe { (*obj).inner.remove(&key).map(|(_, v)| v).unwrap_or(0) }
}

#[no_mangle]
pub extern "C" fn alchemist_map_batch_set(
    obj: *mut AlchemistMap,
    keys: *const usize,
    vals: *const usize,
    len: usize,
) {
    if obj.is_null() || keys.is_null() || vals.is_null() || len == 0 {
        return
    }

    unsafe {
        let key_slice = slice::from_raw_parts(keys, len);
        let val_slice = slice::from_raw_parts(vals, len);
        let map = &(*obj).inner;
        for i in 0..len {
            map.insert(key_slice[i], val_slice[i]);
        }
    }
}

#[no_mangle]
pub extern "C" fn alchemist_map_batch_get(
    obj: *mut AlchemistMap,
    keys: *const usize,
    values_out: *mut usize,
    len: usize,
) {
    if obj.is_null() || keys.is_null() || values_out.is_null() || len == 0 {
        return
    }
    unsafe {
        let keys_slice = slice::from_raw_parts(keys, len);
        let values_slice = slice::from_raw_parts_mut(values_out, len);
        let map = &(*obj).inner;
        for i in 0..len {
            values_slice[i] = map.get(&keys_slice[i])
                .map(|v| *v)
                .unwrap_or(0);
        }
    }
}

#[no_mangle]
pub extern "C" fn alchemist_map_batch_remove(
    obj: *mut AlchemistMap,
    keys: *const usize,
    values_out: *mut usize,
    len: usize,
) {
    if obj.is_null() || keys.is_null() || values_out.is_null() || len == 0 {
        return
    }
    unsafe {
        let keys_slice = slice::from_raw_parts(keys, len);
        let values_slice = slice::from_raw_parts_mut(values_out, len);
        let map = &(*obj).inner;
        for i in 0..len {
            values_slice[i] = map.remove(&keys_slice[i]).map(|(_, v)| v).unwrap_or(0);
        }
    }
}

#[repr(C)]
pub struct AlchemistMapIterator {
    items: Vec<(usize, usize)>,
    pos: usize,
}

#[no_mangle]
pub extern "C" fn alchemist_map_iterator_new(obj: *mut AlchemistMap) -> *mut AlchemistMapIterator {
    if obj.is_null() {
        return std::ptr::null_mut()
    }
    let container = unsafe { &(*obj).inner };
    let mut data = Vec::with_capacity(container.len());
    for kv in container.iter() {
        data.push((*kv.key(), *kv.value()));
    }
    let iterator = AlchemistMapIterator { items: data, pos: 0 };
    Box::into_raw(Box::new(iterator))
}

#[no_mangle]
pub extern "C" fn alchemist_map_iterator_next(
    obj: *mut AlchemistMapIterator,
    key: *mut usize,
    val: *mut usize,
) -> c_int {
    if obj.is_null() || key.is_null() || val.is_null() {
        return 0
    }
    unsafe {
        let iterator = &mut *obj;
        if iterator.pos >= iterator.items.len() {
            return 0
        }
        let (k, v) = iterator.items[iterator.pos];
        *key = k;
        *val = v;
        iterator.pos += 1;
        1
    }
}

#[no_mangle]
pub extern "C" fn alchemist_map_iterator_next_batch(
    obj: *mut AlchemistMapIterator,
    keys_out: *mut usize,
    vals_out: *mut usize,
    max_len: usize,
) -> usize {
    if obj.is_null() || keys_out.is_null() || vals_out.is_null() || max_len == 0 {
        return 0
    }
    let iterator = unsafe { &mut *obj };
    let remaining = iterator.items.len().saturating_sub(iterator.pos);
    let n = remaining.min(max_len);

    if n == 0 {
        return 0
    }

    unsafe {
        let keys_slice = slice::from_raw_parts_mut(keys_out, n);
        let vals_slice = slice::from_raw_parts_mut(vals_out, n);
        for i in 0..n {
            let (k, v) = iterator.items[iterator.pos + i];
            keys_slice[i] = k;
            vals_slice[i] = v;
        }
    }

    iterator.pos += n;
    n
}

#[no_mangle]
pub extern "C" fn alchemist_map_iterator_destroy(obj: *mut AlchemistMapIterator) {
    if !obj.is_null() {
        unsafe { drop(Box::from_raw(obj)) }
    }
}
