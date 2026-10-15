import 'package:quark/utils/file_kind.dart';
import 'package:quark_widgets/quark_widgets.dart' show QuarkModelFormat;

/// The 3D format the file called [name] is read as by the model viewer
/// (#2988, #3225, #3226), by its extension, or null when the viewer does not
/// read it.
QuarkModelFormat? modelFormatForName(String name) =>
    switch (fileExtension(name)) {
      '.stl' => QuarkModelFormat.stl,
      '.obj' => QuarkModelFormat.obj,
      '.ply' => QuarkModelFormat.ply,
      '.gltf' || '.glb' => QuarkModelFormat.gltf,
      '.3mf' => QuarkModelFormat.threeMf,
      _ => null,
    };

/// Whether the file called [name] opens in the 3D model viewer (#2988).
bool opensInModelViewer(String name) => modelFormatForName(name) != null;
